// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
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

package snappy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/progress"

	"gopkg.in/yaml.v2"
)

var (
	errNoSnapToConfig   = errors.New("configuring an invalid snappy package")
	errNoSnapToActivate = errors.New("activating an invalid snappy package")
)

func wrapConfig(pkgName string, conf interface{}) ([]byte, error) {
	configWrap := map[string]map[string]interface{}{
		"config": map[string]interface{}{
			pkgName: conf,
		},
	}

	return yaml.Marshal(configWrap)
}

var newSnapMap = newSnapMapImpl

func newSnapMapImpl() (map[string]*Snap, error) {
	all, err := (&Overlord{}).Installed()
	if err != nil {
		return nil, err
	}

	m := make(map[string]*Snap, 2*len(all))
	for _, snap := range all {
		info := snap.Info()
		m[FullName(info)] = snap
		m[BareName(info)] = snap
	}

	return m, nil
}

type activator interface {
	SetActive(sp *Snap, active bool, meter progress.Meter) error
}

var getActivator = func() activator {
	return &Overlord{}
}

// enableInstalledSnaps activates the installed preinstalled snaps
// on the first boot
func enableInstalledSnaps() error {
	all, err := (&Overlord{}).Installed()
	if err != nil {
		return nil
	}

	activator := getActivator()
	pb := progress.MakeProgressBar()
	for _, sn := range all {
		logger.Noticef("Acitvating %s", FullName(sn.Info()))
		if err := activator.SetActive(sn, true, pb); err != nil {
			// we don't want this to fail for now
			logger.Noticef("failed to activate %s: %s", FullName(sn.Info()), err)
		}
	}

	return nil
}

// FirstBoot checks whether it's the first boot, and if so enables the
// first ethernet device and runs gadgetConfig (as well as flagging that
// it run)
func FirstBoot() error {
	if firstBootHasRun() {
		return ErrNotFirstBoot
	}
	defer stampFirstBoot()
	defer enableFirstEther()

	return enableInstalledSnaps()
}

// NOTE: if you change stampFile, update the condition in
// snapd.firstboot.service to match
var stampFile = "/var/lib/snapd/firstboot/stamp"

func stampFirstBoot() error {
	// filepath.Dir instead of firstbootDir directly to ease testing
	stampDir := filepath.Dir(stampFile)

	if _, err := os.Stat(stampDir); os.IsNotExist(err) {
		if err := os.MkdirAll(stampDir, 0755); err != nil {
			return err
		}
	}

	return osutil.AtomicWriteFile(stampFile, []byte{}, 0644, 0)
}

var globs = []string{"/sys/class/net/eth*", "/sys/class/net/en*"}
var ethdir = "/etc/network/interfaces.d"
var ifup = "/sbin/ifup"

func enableFirstEther() error {
	var eths []string
	for _, glob := range globs {
		eths, _ = filepath.Glob(glob)
		if len(eths) != 0 {
			break
		}
	}
	if len(eths) == 0 {
		return nil
	}
	eth := filepath.Base(eths[0])
	ethfile := filepath.Join(ethdir, eth)
	data := fmt.Sprintf("allow-hotplug %[1]s\niface %[1]s inet dhcp\n", eth)

	if err := osutil.AtomicWriteFile(ethfile, []byte(data), 0644, 0); err != nil {
		return err
	}

	ifup := exec.Command(ifup, eth)
	ifup.Stdout = os.Stdout
	ifup.Stderr = os.Stderr
	if err := ifup.Run(); err != nil {
		return err
	}

	return nil
}

func firstBootHasRun() bool {
	return osutil.FileExists(stampFile)
}
