package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
)

func debugStart(id string) error {
	p := "/tmp/runc-debug"
	if os.Getenv("RUNC_DEBUG_PATH") != "" {
		p = os.Getenv("RUNC_DEBUG_PATH")
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(p, fmt.Sprintf("%v-%v.log", id, time.Now().UnixNano())), os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	l := logrus.New()
	l.Out = f
	l.Level = logrus.DebugLevel

	l.Debugf("args: %v", os.Args)
	l.Debugf("env: %#v", os.Environ())
	groups, err := os.Getgroups()
	l.Debugf("uid: %v, gid: %v, groups: %v, %v", os.Getuid(), os.Getgid(), groups, err)
	wd, err := os.Getwd()
	l.Debugf("cwd: %v, %v", wd, err)
	host, err := os.Hostname()
	l.Debugf("host: %v, %v", host, err)

	config, err := ioutil.ReadFile("./config.json")
	l.Debugf("config: %v", err)
	if err != nil {
		return err
	}

	fmt.Fprintf(f, "%s\n", config)

	var c struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
		Mounts []struct {
			Type   string `json:"type"`
			Source string `json:"source"`
		} `json:"mounts"`
	}

	if err := json.Unmarshal(config, &c); err != nil {
		return err
	}

	debugPath(l, "root", c.Root.Path)
	for i, m := range c.Mounts {
		if m.Type == "bind" {
			debugPath(l, fmt.Sprintf("mount-%d %s", i, m.Source), m.Source)
		}
	}

	return nil
}

func debugPath(l *logrus.Logger, name, p string) {
	fi, err := os.Lstat(p)
	if err != nil {
		l.Errorf("lstat %v: %v", name, err)
		return
	}
	l.Debugf("lstat %v: %v %v %v %v %#v", name, fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir(), fi.Sys())
	if fi.Mode()&os.ModeSymlink != 0 {
		rl, err := os.Readlink(p)
		if err != nil {
			l.Errorf("rl %v: %v", name, err)
		}
		fi, err = os.Lstat(rl)
		if err != nil {
			l.Errorf("stat %v: %v", name, err)
			return
		}
		l.Debugf("stat %v: %v %v %v %v %#v", fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir(), fi.Sys())
	}
	if fi.IsDir() {
		f, err := os.Open(p)
		if err != nil {
			l.Errorf("open %v: %v", name, err)
			return
		}
		fis, err := f.Readdir(-1)
		if err != nil {
			l.Errorf("readdir %v: %v", name, err)
			return
		}
		for _, cfi := range fis {
			l.Errorf("readdir: %v, %v, %v", cfi.Name(), cfi.IsDir(), cfi.Mode())
		}
	}
}
