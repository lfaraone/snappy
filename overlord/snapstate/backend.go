// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package snapstate

import (
	"github.com/snapcore/snapd/overlord/snapstate/backend"
	"github.com/snapcore/snapd/progress"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/store"
)

// A StoreService can find, list available updates and offer for download snaps.
type StoreService interface {
	Snap(string, string, store.Authenticator) (*snap.Info, error)
	Find(string, string, store.Authenticator) ([]*snap.Info, error)
	ListRefresh([]*store.RefreshCandidate, store.Authenticator) ([]*snap.Info, error)
	SuggestedCurrency() string

	Download(*snap.Info, progress.Meter, store.Authenticator) (string, error)
}

type managerBackend interface {
	// install releated
	Download(name, channel string, checker func(*snap.Info) error, meter progress.Meter, store StoreService, auther store.Authenticator) (*snap.Info, string, error)
	SetupSnap(snapFilePath string, si *snap.SideInfo, meter progress.Meter) error
	CopySnapData(newSnap, oldSnap *snap.Info, meter progress.Meter) error
	LinkSnap(info *snap.Info) error
	// the undoers for install
	UndoSetupSnap(s snap.PlaceInfo, meter progress.Meter) error
	UndoCopySnapData(newSnap, oldSnap *snap.Info, meter progress.Meter) error

	// remove releated
	UnlinkSnap(info *snap.Info, meter progress.Meter) error
	RemoveSnapFiles(s snap.PlaceInfo, meter progress.Meter) error
	RemoveSnapData(info *snap.Info) error
	RemoveSnapCommonData(info *snap.Info) error

	// testing helpers
	Current(cur *snap.Info)
	Candidate(sideInfo *snap.SideInfo)
}

type defaultBackend struct {
	// XXX defaultBackend will go away and be replaced by this in the end.
	backend.Backend
}

func (b *defaultBackend) Candidate(*snap.SideInfo) {}
func (b *defaultBackend) Current(*snap.Info)       {}

func (b *defaultBackend) Download(name, channel string, checker func(*snap.Info) error, meter progress.Meter, stor StoreService, auther store.Authenticator) (*snap.Info, string, error) {
	snap, err := stor.Snap(name, channel, auther)
	if err != nil {
		return nil, "", err
	}

	err = checker(snap)
	if err != nil {
		return nil, "", err
	}

	downloadedSnapFile, err := stor.Download(snap, meter, auther)
	if err != nil {
		return nil, "", err
	}

	return snap, downloadedSnapFile, nil
}
