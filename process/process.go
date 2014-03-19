// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package process

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

const (
	defaultUser  = "nobody"
	defaultGroup = "nobody"
)

// Init create pid file, set working dir, setgid and setuid.
func Init(userGroup, dir, pidFile string) error {
	// setuid and setgid
	ug := strings.SplitN(userGroup, " ", 2)
	usr := defaultUser
	grp := defaultGroup
	if len(ug) == 0 {
		// default user and group (nobody)
	} else if len(ug) == 1 {
		usr = ug[0]
		grp = ""
	} else if len(ug) == 2 {
		usr = ug[0]
		grp = ug[1]
	}
	uid := 0
	gid := 0
	ui, err := user.Lookup(usr)
	if err != nil {
		return err
	}
	uid, _ = strconv.Atoi(ui.Uid)
	// group no set
	if grp == "" {
		gid, _ = strconv.Atoi(ui.Gid)
	} else {
		// use user's group instread
		// TODO LookupGroup
		gid, _ = strconv.Atoi(ui.Gid)
	}
	if err := syscall.Setgid(gid); err != nil {
		return err
	}
	if err := syscall.Setuid(uid); err != nil {
		return err
	}
	// change working dir
	if err := os.Chdir(dir); err != nil {
		return err
	}
	// create pid file
	if err := ioutil.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644); err != nil {
		return err
	}
	return nil
}
