package main

import (
	"C"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	sshserver "github.com/gliderlabs/ssh"
	"github.com/kr/pty"
	"github.com/sirupsen/logrus"
)

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <unistd.h>
#include <crypt.h>
#include <shadow.h>
#include <string.h>
*/
import "C"
import (
	"os/user"
	"strconv"
)

func Auth(user string, passwd string) bool {
	cuser := C.CString(user)
	defer C.free(unsafe.Pointer(cuser))

	cpasswd := C.CString(passwd)
	defer C.free(unsafe.Pointer(cpasswd))

	pwd := C.getspnam(cuser)
	if pwd == nil {
		return false
	}

	crypted := C.crypt(cpasswd, pwd.sp_pwdp)

	if C.strcmp(crypted, pwd.sp_pwdp) != 0 {
		return false
	}

	return true
}

type SSHServer struct {
	sshd *sshserver.Server
}

func NewSSHServer(port int) *SSHServer {
	s := &SSHServer{}

	s.sshd = &sshserver.Server{
		Addr: fmt.Sprintf("localhost:%d", port),
		PasswordHandler: func(ctx sshserver.Context, pass string) bool {
			if Auth(ctx.User(), pass) == true {
				return true
			}

			return false
		},
		PublicKeyHandler: s.publicKeyHandler,
		Handler:          s.sessionHandler,
	}

	return s
}

func (s *SSHServer) ListenAndServe() error {
	return s.sshd.ListenAndServe()
}

func (s *SSHServer) sessionHandler(session sshserver.Session) {
	sspty, winCh, isPty := session.Pty()

	if isPty {
		scmd := newShellCmd(session.User(), sspty.Term)

		spty, err := pty.Start(scmd)
		if err != nil {
			logrus.Warn(err)
		}

		go func() {
			for win := range winCh {
				setWinsize(spty, win.Width, win.Height)
			}
		}()

		go func() {
			_, err := io.Copy(session, spty)
			if err != nil {
				logrus.Warn(err)
			}
		}()

		go func() {
			_, err := io.Copy(spty, session)
			if err != nil {
				logrus.Warn(err)
			}
		}()

		err = scmd.Wait()
		if err != nil {
			logrus.Warn(err)
		}
	} else {
		cmd := exec.Command(session.Command()[0], session.Command()[1:]...)

		stdout, _ := cmd.StdoutPipe()
		stdin, _ := cmd.StdinPipe()

		cmd.Start()

		go func() {
			if _, err := io.Copy(stdin, session); err != nil {
				fmt.Println(err)
			}
		}()

		go func() {
			if _, err := io.Copy(session, stdout); err != nil {
				fmt.Println(err)
			}
		}()

		cmd.Wait()
	}
}

func (s *SSHServer) publicKeyHandler(ctx sshserver.Context, key sshserver.PublicKey) bool {
	return true
}

func newShellCmd(username string, term string) *exec.Cmd {
	shell := os.Getenv("SHELL")

	if shell == "" {
		shell = "/bin/sh"
	}

	if term == "" {
		term = "xterm"
	}

	u, _ := user.Lookup(username)
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	cmd := exec.Command(shell, "--login")
	cmd.Env = []string{
		"TERM=" + term,
		"HOME=" + u.HomeDir,
		"SHELL=" + shell,
	}
	cmd.Dir = u.HomeDir
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

	return cmd
}

func setWinsize(f *os.File, w, h int) {
	size := &struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0}
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(size)))
}
