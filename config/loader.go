package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

var reKeybase = regexp.MustCompile(`keybase@([a-z][a-z0-9]*):\/\/((?:team|private|public).*)`)

func readConfig(logger *zap.Logger, p string) (data []byte, err error) {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "keybase") {
		matches := reKeybase.FindStringSubmatch(p)
		if len(matches) == 0 {
			return nil, fmt.Errorf("bad KBFS config: %q. KBFS paths should be in the format of keybase@<system-username>://team|private|public/...", p)
		}
		kbfsPath := "keybase://" + matches[2]
		username := matches[1]
		logger.Sugar().Debugf("reading config at KBFS path %q as user %s", kbfsPath, username)

		u, err := user.Lookup(username)
		if err != nil {
			return nil, err
		}
		// TODO: u.Uid and u.Gid are not decimal numbers on windows
		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return nil, err
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			return nil, err
		}

		cmd := exec.Command("keybase", "fs", "read", kbfsPath)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid:         uint32(uid),
				Gid:         uint32(gid),
				NoSetGroups: true,
			},
		}
		return cmd.Output()
	} else if strings.HasPrefix(p, "https://") {
		logger.Sugar().Debugf("reading config at URL %s", p)
		resp, err := http.Get(p)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	} else {
		logger.Sugar().Debugf("reading config at filesystem path %s", p)
		return ioutil.ReadFile(p)
	}
}
