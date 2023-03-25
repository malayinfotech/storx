// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

// Package admin implements administrative endpoints for satellite.
package admin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"common/errs2"
	"storx/satellite/accounting"
	adminui "storx/satellite/admin/ui"
	"storx/satellite/buckets"
	"storx/satellite/console"
	"storx/satellite/console/consoleweb"
	"storx/satellite/console/restkeys"
	"storx/satellite/oidc"
	"storx/satellite/payments"
	"storx/satellite/payments/stripecoinpayments"
)

// Config defines configuration for debug server.
type Config struct {
	Address          string `help:"admin peer http listening address" releaseDefault:"" devDefault:""`
	StaticDir        string `help:"an alternate directory path which contains the static assets to serve. When empty, it uses the embedded assets" releaseDefault:"" devDefault:""`
	AllowedOauthHost string `help:"the oauth host allowed to bypass token authentication."`
	Groups           Groups

	AuthorizationToken string `internal:"true"`
}

// Groups defines permission groups.
type Groups struct {
	LimitUpdate string `help:"the group which is only allowed to update user and project limits and freeze and unfreeze accounts."`
}

// DB is databases needed for the admin server.
type DB interface {
	// ProjectAccounting returns database for storing information about project data use
	ProjectAccounting() accounting.ProjectAccounting
	// Console returns database for satellite console
	Console() console.DB
	// OIDC returns the database for OIDC and OAuth information.
	OIDC() oidc.DB
	// StripeCoinPayments returns database for satellite stripe coin payments
	StripeCoinPayments() stripecoinpayments.DB
}

// Server provides endpoints for administrative tasks.
type Server struct {
	log *zap.Logger

	listener net.Listener
	server   http.Server

	db             DB
	payments       payments.Accounts
	buckets        *buckets.Service
	restKeys       *restkeys.Service
	freezeAccounts *console.AccountFreezeService

	nowFn func() time.Time

	console consoleweb.Config
	config  Config
}

// NewServer returns a new administration Server.
func NewServer(log *zap.Logger, listener net.Listener, db DB, buckets *buckets.Service, restKeys *restkeys.Service, freezeAccounts *console.AccountFreezeService, accounts payments.Accounts, console consoleweb.Config, config Config) *Server {
	server := &Server{
		log: log,

		listener: listener,

		db:             db,
		payments:       accounts,
		buckets:        buckets,
		restKeys:       restKeys,
		freezeAccounts: freezeAccounts,

		nowFn: time.Now,

		console: console,
		config:  config,
	}

	root := mux.NewRouter()

	api := root.PathPrefix("/api/").Subrouter()

	// When adding new options, also update README.md

	// prod owners only
	fullAccessAPI := api.NewRoute().Subrouter()
	fullAccessAPI.Use(withAuth(log, config, nil))
	fullAccessAPI.HandleFunc("/users", server.addUser).Methods("POST")
	fullAccessAPI.HandleFunc("/users/{useremail}", server.updateUser).Methods("PUT")
	fullAccessAPI.HandleFunc("/users/{useremail}", server.deleteUser).Methods("DELETE")
	fullAccessAPI.HandleFunc("/users/{useremail}/mfa", server.disableUserMFA).Methods("DELETE")
	fullAccessAPI.HandleFunc("/oauth/clients", server.createOAuthClient).Methods("POST")
	fullAccessAPI.HandleFunc("/oauth/clients/{id}", server.updateOAuthClient).Methods("PUT")
	fullAccessAPI.HandleFunc("/oauth/clients/{id}", server.deleteOAuthClient).Methods("DELETE")
	fullAccessAPI.HandleFunc("/projects", server.addProject).Methods("POST")
	fullAccessAPI.HandleFunc("/projects/{project}", server.renameProject).Methods("PUT")
	fullAccessAPI.HandleFunc("/projects/{project}", server.deleteProject).Methods("DELETE")
	fullAccessAPI.HandleFunc("/projects/{project}", server.getProject).Methods("GET")
	fullAccessAPI.HandleFunc("/projects/{project}/apikeys", server.addAPIKey).Methods("POST")
	fullAccessAPI.HandleFunc("/projects/{project}/apikeys", server.listAPIKeys).Methods("GET")
	fullAccessAPI.HandleFunc("/projects/{project}/apikeys/{name}", server.deleteAPIKeyByName).Methods("DELETE")
	fullAccessAPI.HandleFunc("/projects/{project}/buckets/{bucket}", server.getBucketInfo).Methods("GET")
	fullAccessAPI.HandleFunc("/projects/{project}/buckets/{bucket}/geofence", server.createGeofenceForBucket).Methods("POST")
	fullAccessAPI.HandleFunc("/projects/{project}/buckets/{bucket}/geofence", server.deleteGeofenceForBucket).Methods("DELETE")
	fullAccessAPI.HandleFunc("/projects/{project}/usage", server.checkProjectUsage).Methods("GET")
	fullAccessAPI.HandleFunc("/apikeys/{apikey}", server.deleteAPIKey).Methods("DELETE")
	fullAccessAPI.HandleFunc("/restkeys/{useremail}", server.addRESTKey).Methods("POST")
	fullAccessAPI.HandleFunc("/restkeys/{apikey}/revoke", server.revokeRESTKey).Methods("PUT")

	// limit update access required
	limitUpdateAPI := api.NewRoute().Subrouter()
	limitUpdateAPI.Use(withAuth(log, config, []string{config.Groups.LimitUpdate}))
	limitUpdateAPI.HandleFunc("/users/{useremail}", server.userInfo).Methods("GET")
	limitUpdateAPI.HandleFunc("/users/{useremail}/limits", server.userLimits).Methods("GET")
	limitUpdateAPI.HandleFunc("/users/{useremail}/limits", server.updateLimits).Methods("PUT")
	limitUpdateAPI.HandleFunc("/users/{useremail}/freeze", server.freezeUser).Methods("PUT")
	limitUpdateAPI.HandleFunc("/users/{useremail}/freeze", server.unfreezeUser).Methods("DELETE")
	limitUpdateAPI.HandleFunc("/projects/{project}/limit", server.getProjectLimit).Methods("GET")
	limitUpdateAPI.HandleFunc("/projects/{project}/limit", server.putProjectLimit).Methods("PUT", "POST")

	// This handler must be the last one because it uses the root as prefix,
	// otherwise will try to serve all the handlers set after this one.
	if config.StaticDir == "" {
		root.PathPrefix("/").Handler(http.FileServer(http.FS(adminui.Assets))).Methods("GET")
	} else {
		root.PathPrefix("/").Handler(http.FileServer(http.Dir(config.StaticDir))).Methods("GET")
	}

	server.server.Handler = root
	return server
}

// Run starts the admin endpoint.
func (server *Server) Run(ctx context.Context) error {
	if server.listener == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	var group errgroup.Group
	group.Go(func() error {
		<-ctx.Done()
		return Error.Wrap(server.server.Shutdown(context.Background()))
	})
	group.Go(func() error {
		defer cancel()
		err := server.server.Serve(server.listener)
		if errs2.IsCanceled(err) || errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		return Error.Wrap(err)
	})
	return group.Wait()
}

// SetNow allows tests to have the server act as if the current time is whatever they want.
func (server *Server) SetNow(nowFn func() time.Time) {
	server.nowFn = nowFn
}

// Close closes server and underlying listener.
func (server *Server) Close() error {
	return Error.Wrap(server.server.Close())
}

// withAuth checks if the requester is authorized to perform an operation. If the request did not come from the oauth proxy, verify the auth token.
// Otherwise, check that the user has the required permissions to conduct the operation. `allowedGroups` is a list of groups that are authorized.
// If it is nil, then the api method is not accessible from the oauth proxy.
func withAuth(log *zap.Logger, config Config, allowedGroups []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Host != config.AllowedOauthHost {
				// not behind the proxy; use old authentication method.
				if config.AuthorizationToken == "" {
					sendJSONError(w, "Authorization not enabled.",
						"", http.StatusForbidden)
					return
				}

				equality := subtle.ConstantTimeCompare(
					[]byte(r.Header.Get("Authorization")),
					[]byte(config.AuthorizationToken),
				)
				if equality != 1 {
					sendJSONError(w, "Forbidden",
						"", http.StatusForbidden)
					return
				}
			} else {
				// request made from oauth proxy. Check user groups against allowedGroups.
				if allowedGroups == nil {
					sendJSONError(w, "Forbidden",
						"This operation is not authorized through oauth. Please contact a production owner to proceed.", http.StatusForbidden)
					return
				}

				var allowed bool
				userGroupsString := r.Header.Get("X-Forwarded-Groups")
				userGroups := strings.Split(userGroupsString, ",")
				for _, userGroup := range userGroups {
					if userGroup == "" {
						continue
					}
					for _, permGroup := range allowedGroups {
						if userGroup == permGroup {
							allowed = true
							break
						}
					}
					if allowed {
						break
					}
				}

				if !allowed {
					sendJSONError(w, "Forbidden",
						fmt.Sprintf("User must be a member of one of these groups to conduct this operation: %s", allowedGroups), http.StatusForbidden)
					return
				}
			}

			log.Info(
				"admin action",
				zap.String("host", r.Host),
				zap.String("user", r.Header.Get("X-Forwarded-Email")),
				zap.String("action", fmt.Sprintf("%s-%s", r.Method, r.RequestURI)),
				zap.String("queries", r.URL.Query().Encode()),
			)

			r.Header.Set("Cache-Control", "must-revalidate")
			next.ServeHTTP(w, r)
		})
	}
}
