// Copyright (c) 2022 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tka

import (
	"fmt"

	"tailscale.com/types/tkatype"
)

// Types implementing Signer can sign update messages.
type Signer interface {
	SignAUM(tkatype.AUMSigHash) ([]tkatype.Signature, error)
}

// UpdateBuilder implements a builder for changes to the tailnet
// key authority.
//
// Finalize must be called to compute the update messages, which
// must then be applied to all Authority objects using Inform().
type UpdateBuilder struct {
	a      *Authority
	signer Signer

	state  State
	parent AUMHash

	out []AUM
}

func (b *UpdateBuilder) mkUpdate(update AUM) error {
	prevHash := make([]byte, len(b.parent))
	copy(prevHash, b.parent[:])
	update.PrevAUMHash = prevHash

	if b.signer != nil {
		sigs, err := b.signer.SignAUM(update.SigHash())
		if err != nil {
			return fmt.Errorf("signing failed: %v", err)
		}
		update.Signatures = append(update.Signatures, sigs...)
	}
	if err := update.StaticValidate(); err != nil {
		return fmt.Errorf("generated update was invalid: %v", err)
	}
	state, err := b.state.applyVerifiedAUM(update)
	if err != nil {
		return fmt.Errorf("update cannot be applied: %v", err)
	}

	b.state = state
	b.parent = update.Hash()
	b.out = append(b.out, update)
	return nil
}

// AddKey adds a new key to the authority.
func (b *UpdateBuilder) AddKey(key Key) error {
	if _, err := b.state.GetKey(key.ID()); err == nil {
		return fmt.Errorf("cannot add key %v: already exists", key)
	}
	return b.mkUpdate(AUM{MessageKind: AUMAddKey, Key: &key})
}

// RemoveKey removes a key from the authority.
func (b *UpdateBuilder) RemoveKey(keyID tkatype.KeyID) error {
	if _, err := b.state.GetKey(keyID); err != nil {
		return fmt.Errorf("failed reading key %x: %v", keyID, err)
	}
	return b.mkUpdate(AUM{MessageKind: AUMRemoveKey, KeyID: keyID})
}

// SetKeyVote updates the number of votes of an existing key.
func (b *UpdateBuilder) SetKeyVote(keyID tkatype.KeyID, votes uint) error {
	if _, err := b.state.GetKey(keyID); err != nil {
		return fmt.Errorf("failed reading key %x: %v", keyID, err)
	}
	return b.mkUpdate(AUM{MessageKind: AUMUpdateKey, Votes: &votes, KeyID: keyID})
}

// SetKeyMeta updates key-value metadata stored against an existing key.
//
// TODO(tom): Provide an API to update specific values rather than the whole
// map.
func (b *UpdateBuilder) SetKeyMeta(keyID tkatype.KeyID, meta map[string]string) error {
	if _, err := b.state.GetKey(keyID); err != nil {
		return fmt.Errorf("failed reading key %x: %v", keyID, err)
	}
	return b.mkUpdate(AUM{MessageKind: AUMUpdateKey, Meta: meta, KeyID: keyID})
}

// Finalize returns the set of update message to actuate the update.
func (b *UpdateBuilder) Finalize() ([]AUM, error) {
	if len(b.out) > 0 {
		if parent, _ := b.out[0].Parent(); parent != b.a.Head() {
			return nil, fmt.Errorf("updates no longer apply to head: based on %x but head is %x", parent, b.a.Head())
		}
	}
	return b.out, nil
}

// NewUpdater returns a builder you can use to make changes to
// the tailnet key authority.
//
// The provided signer function, if non-nil, is called with each update
// to compute and apply signatures.
//
// Updates are specified by calling methods on the returned UpdatedBuilder.
// Call Finalize() when you are done to obtain the specific update messages
// which actuate the changes.
func (a *Authority) NewUpdater(signer Signer) *UpdateBuilder {
	return &UpdateBuilder{
		a:      a,
		signer: signer,
		parent: a.Head(),
		state:  a.state,
	}
}
