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
	// change working dir
	if err := os.Chdir(Conf.Dir); err != nil {
		Log.Error("os.Chdir(\"%s\") failed (%s)", "", err.Error())
		return err
	}
	// create pid file
	if err := ioutil.WriteFile(Conf.PidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644); err != nil {
		Log.Error("ioutil.WriteFile(\"%s\") failed (%s)", "", err.Error())
		return err
	}
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
		Log.Error("user.Lookup(\"%s\") failed (%s)", err.Error())
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
	if err := syscall.Setuid(uid); err != nil {
		Log.Error("syscall.Setuid(%d) failed (%s)", uid, err.Error())
		return err
	}
	if err := syscall.Setgid(gid); err != nil {
		Log.Error("syscall.Setgid(%d) failed (%s)", gid, err.Error())
		return err
	}
	return nil
}
