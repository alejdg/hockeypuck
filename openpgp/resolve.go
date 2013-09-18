/*
   Hockeypuck - OpenPGP key server
   Copyright (C) 2012, 2013  Casey Marshall

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, version 3.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package openpgp

import (
	_ "code.google.com/p/go.crypto/md4"
	_ "code.google.com/p/go.crypto/ripemd160"
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

type resolver struct {
	Pubkey *Pubkey
}

// Resolve resolves and connects relationship references
// between the different packet records in the key material.
func Resolve(pubkey *Pubkey) {
	r := &resolver{pubkey}
	var signable Signable
	scopedPackets := make(map[string]bool)
	pubkey.Visit(func(rec PacketRecord) error {
		switch p := rec.(type) {
		case *Pubkey:
			r.setSigScope(p.RFingerprint, p.signatures...)
			p.linkSelfSigs()
			signable = p
		case *UserId:
			p.ScopedDigest = p.calcScopedDigest(r.Pubkey)
			if _, has := scopedPackets[p.ScopedDigest]; has {
				r.Pubkey.userIds = removeUserId(r.Pubkey.userIds, p)
				r.Pubkey.Unsupported = append(r.Pubkey.Unsupported, p.Packet...)
				r.Pubkey.Unsupported = append(r.Pubkey.Unsupported, concatSigPackets(p.signatures)...)
				p.signatures = nil
			} else {
				scopedPackets[p.ScopedDigest] = true
				r.setSigScope(p.ScopedDigest, p.signatures...)
				p.linkSelfSigs(r.Pubkey)
				signable = p
				// linkSelfSigs needs to set creation & expiration
			}
		case *UserAttribute:
			p.ScopedDigest = p.calcScopedDigest(r.Pubkey)
			if _, has := scopedPackets[p.ScopedDigest]; has {
				r.Pubkey.userAttributes = removeUserAttribute(r.Pubkey.userAttributes, p)
				r.Pubkey.Unsupported = append(r.Pubkey.Unsupported, p.Packet...)
				r.Pubkey.Unsupported = append(r.Pubkey.Unsupported, concatSigPackets(p.signatures)...)
				p.signatures = nil
			} else {
				scopedPackets[p.ScopedDigest] = true
				r.setSigScope(p.ScopedDigest, p.signatures...)
				p.linkSelfSigs(r.Pubkey)
				signable = p
				// linkSelfSigs needs to set creation & expiration
			}
		case *Subkey:
			r.setSigScope(p.RFingerprint, p.signatures...)
			p.linkSelfSigs(r.Pubkey)
			signable = p
		case *Signature:
			if _, has := scopedPackets[p.ScopedDigest]; has {
				signable.RemoveSignature(p)
				r.Pubkey.Unsupported = append(r.Pubkey.Unsupported, p.Packet...)
			} else {
				scopedPackets[p.ScopedDigest] = true
			}
		}
		return nil
	})
}

func (r *resolver) setSigScope(scope string, sigs ...*Signature) {
	for _, sig := range sigs {
		sig.ScopedDigest = sig.calcScopedDigest(r.Pubkey, scope)
	}
}

func removeUserId(uids []*UserId, removeUid *UserId) (result []*UserId) {
	for _, uid := range uids {
		if removeUid != uid {
			result = append(result, uid)
		}
	}
	return
}

func removeUserAttribute(uats []*UserAttribute, removeUat *UserAttribute) (result []*UserAttribute) {
	for _, uat := range uats {
		if removeUat != uat {
			result = append(result, uat)
		}
	}
	return
}

func removeSignature(sigs []*Signature, removeSig *Signature) (result []*Signature) {
	for _, sig := range sigs {
		if removeSig != sig {
			result = append(result, sig)
		}
	}
	return
}

func concatSigPackets(sigs []*Signature) (result []byte) {
	for _, sig := range sigs {
		result = append(result, sig.Packet...)
	}
	return
}
