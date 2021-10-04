// Package external implements functionality for communicating and parsing external or 3rd party data sources.
package external

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
)

var (
	networks []*net.IPNet
	steamids []steamid.SID64
)

func containsSID(sid steamid.SID64) bool {
	for _, s := range steamids {
		if s.Int64() == sid.Int64() {
			return true
		}
	}
	return false
}

func containsIP(ip net.IP) bool {
	for _, b := range networks {
		if b.Contains(ip) {
			return true
		}
	}
	return false
}

// Import is used to download and load block lists into memory
func Import(list config.BanList) error {
	if !golib.Exists(config.Net.CachePath) {
		if err := os.MkdirAll(config.Net.CachePath, 0755); err != nil {
			log.Fatalf("Failed to create cache dir (%s): %v", config.Net.CachePath, err)
		}
	}
	filePath := path.Join(config.Net.CachePath, list.Name)
	maxAge, err := config.ParseDuration(config.Net.MaxAge)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse cache max age")
	}
	expired := false
	if golib.Exists(filePath) {
		f, err := os.Stat(filePath)
		if err != nil {
			return errors.Wrapf(err, "Failed to stat cached file")
		}
		if config.Now().Sub(f.ModTime()) > maxAge {
			expired = true
		}
	} else {
		expired = true
	}
	if expired {
		if err := download(list.URL, filePath); err != nil {
			return errors.Wrapf(err, "Failed to download net ban list")
		}
	}
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	cnt, err := load(b, list.Type)
	if err != nil {
		return errors.Wrapf(err, "Failed to load list")
	}
	log.Infof("Loaded %d blocks from %s", cnt, list.Name)
	return nil
}

func download(url string, savePath string) error {
	resp, err := util.NewHTTPClient().Get(url)
	if err != nil {
		return err
	}
	outFile, err := os.Create(savePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close block list response body: %v", err)
		}
	}()
	return nil
}

func load(src []byte, listType config.BanListType) (count int, err error) {
	switch listType {
	case config.CIDR:
		nets, err := parseCIDR(src)
		if err != nil {
			return 0, err
		}
		return addNets(nets), nil
	case config.ValveNet:
		nets, err := parseValveNet(src)
		if err != nil {
			return 0, err
		}
		return addNets(nets), nil
	case config.ValveSID:
		ids, err := parseValveSID(src)
		if err != nil {
			return 0, err
		}
		return addSIDs(ids), nil
	case config.TF2BD:
		ids, err := parseTF2BD(src)
		if err != nil {
			return 0, err
		}
		return addSIDs(ids), nil
	default:
		return 0, errors.Errorf("Unimplemented list type: %v", listType)
	}
}

func addNets(nets []*net.IPNet) int {
	cnt := 0
	for _, n := range nets {
		if !containsIP(n.IP) {
			networks = append(networks, n)
			cnt++
		}
	}
	return cnt
}

func addSIDs(sids []steamid.SID64) int {
	cnt := 0
	for _, s := range sids {
		if !containsSID(s) {
			steamids = append(steamids, s)
			cnt++
		}
	}
	return cnt
}

func parseCIDR(src []byte) ([]*net.IPNet, error) {
	var n []*net.IPNet
	for _, line := range strings.Split(string(src), "\n") {
		if line == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(line)
		if err != nil {
			continue
		}
		n = append(n, ipNet)
	}
	return n, nil
}
