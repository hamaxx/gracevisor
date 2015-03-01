// Copied from https://github.com/bradfitz/runsit so copyright applies. Check LICENSE.deps for details.
// Modified by jure@hamsworld.net

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

type GvCmd struct {
	Uid  int // or 0 to not change
	Gid  int // or 0 to not change
	Path string
	Env  []string
	Argv []string
	Dir  string
}

func (c *GvCmd) start() (cmd *exec.Cmd, outPipe, errPipe io.ReadCloser, err error) {
	var buf bytes.Buffer
	b64enc := base64.NewEncoder(base64.StdEncoding, &buf)
	err = gob.NewEncoder(b64enc).Encode(c)
	b64enc.Close()
	if err != nil {
		return
	}

	cmd = exec.Command(os.Args[0])
	cmd.Env = append(cmd.Env, "_GRACEVISOR_LAUNCH_INFO="+buf.String())
	// TODO catch term signal and manually stop instances
	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Setpgid: true,
	//}

	outPipe, err = cmd.StdoutPipe()
	if err != nil {
		return
	}
	errPipe, err = cmd.StderrPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	return cmd, outPipe, errPipe, nil
}

func MaybeBecomeChildProcess() {
	cs := os.Getenv("_GRACEVISOR_LAUNCH_INFO")
	if cs == "" {
		return
	}
	defer os.Exit(2)

	c := new(GvCmd)
	d := gob.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(cs)))
	err := d.Decode(c)
	if err != nil {
		log.Fatalf("Failed to decode GvCmd in child: %v", err)
	}

	// Setuid and Setgid work only for the current thread so we want to make sure to lock it
	runtime.LockOSThread()

	if c.Gid != 0 {
		if err := Setgid(c.Gid); err != nil {
			log.Fatalf("failed to Setgid(%d): %v", c.Gid, err)
		}
	}
	if c.Uid != 0 {
		if err := Setuid(c.Uid); err != nil {
			log.Fatalf("failed to Setuid(%d): %v", c.Uid, err)
		}
	}
	if c.Dir != "" {
		err = os.Chdir(c.Dir)
		if err != nil {
			log.Fatalf("failed to chdir to %q: %v", c.Dir, err)
		}
	}
	err = syscall.Exec(c.Path, c.Argv, c.Env)
	log.Fatalf("failed to exec %q: %v", c.Path, err)
}
