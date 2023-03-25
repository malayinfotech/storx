// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"common/fpath"
	"private/cfgstruct"
	"private/process"
	_ "storx/private/version" // This attaches version information during release builds.
)

var (
	rootCmd = &cobra.Command{
		Use:   "storx-admin",
		Short: "A tool for managing operations against a satellite",
	}
	setupCmd = &cobra.Command{
		Use:         "setup",
		Short:       "Create config files",
		RunE:        cmdSetup,
		Annotations: map[string]string{"type": "setup"},
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the storx-admin",
	}
	confDir string

	runCfg   AdminConf
	setupCfg AdminConf
)

// AdminConf defines necessary configuration to run the storx-admin UI.
type AdminConf struct {
	AuthKey     string `help:"API authorization key" default:""`
	Address     string `help:"address to start the web server on" default:":8080"`
	EndpointURL string `help:"satellite admin endpoint" default:"localhost:7778"`
}

func cmdSetup(cmd *cobra.Command, args []string) (err error) {
	setupDir, err := filepath.Abs(confDir)
	if err != nil {
		return err
	}

	valid, _ := fpath.IsValidSetupDir(setupDir)
	if !valid {
		return fmt.Errorf("satellite configuration already exists (%v)", setupDir)
	}

	err = os.MkdirAll(setupDir, 0700)
	if err != nil {
		return err
	}

	return process.SaveConfig(cmd, filepath.Join(setupDir, "config.yaml"))
}

func init() {
	defaultConfDir := fpath.ApplicationDir("storx", "storx-admin")
	cfgstruct.SetupFlag(zap.L(), rootCmd, &confDir, "config-dir", defaultConfDir, "main directory for satellite configuration")
	defaults := cfgstruct.DefaultsFlag(rootCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(setupCmd)
	process.Bind(runCmd, &runCfg, defaults, cfgstruct.ConfDir(confDir))
	process.Bind(setupCmd, &setupCfg, defaults, cfgstruct.ConfDir(confDir), cfgstruct.SetupMode())
}

func main() {
	process.Exec(rootCmd)
}
