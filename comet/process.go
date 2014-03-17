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

package main

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

// InitProcess create pid file, set working dir, setgid and setuid.
func InitProcess() error {
	// setuid and setgid
	ug := strings.SplitN(Conf.User, " ", 2)
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
		Log.Error("user.Lookup(\"%s\") error(%v)", err)
		return err
	}
	uid, _ = strconv.Atoi(ui.Uid)
	// group no set
	if grp == "" {
		Log.Debug("no set group")
		gid, _ = strconv.Atoi(ui.Gid)
	} else {
		// use user's group instread
		// TODO LookupGroup
		gid, _ = strconv.Atoi(ui.Gid)
	}
	Log.Debug("set user: %d", uid)
	if err := syscall.Setuid(uid); err != nil {
		Log.Error("syscall.Setuid(%d) error(%v)", uid, err)
		return err
	}
	//if err := syscall.Setgid(gid); err != nil {
	//	Log.Error("syscall.Setgid(%d) failed (%s)", gid, err.Error())
	//	return err
	//}
	// change working dir
	Log.Debug("set gid: %d", gid)
	if err := os.Chdir(Conf.Dir); err != nil {
		Log.Error("os.Chdir(\"%s\") error(%v)", "", err)
		return err
	}
	// create pid file
	if err := ioutil.WriteFile(Conf.PidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644); err != nil {
		Log.Error("ioutil.WriteFile(\"%s\") error(%v)", "", err)
		return err
	}
	return nil
}
