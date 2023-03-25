// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information

package simulate

import (
	"context"
	"net/mail"

	"storx/private/post"
)

// NoMail doesn't send out any mail.
type NoMail struct {
}

// SendEmail implements func from mailservice.Sender.
func (n NoMail) SendEmail(ctx context.Context, msg *post.Message) error {
	return nil
}

// FromAddress implements func from mailservice.Sender.
func (n NoMail) FromAddress() post.Address {
	addr, _ := mail.ParseAddress("nosuchmail@storx")
	return *addr
}
