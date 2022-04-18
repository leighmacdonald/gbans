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
		if errMkDir := os.MkdirAll(config.Net.CachePath, 0755); errMkDir != nil {
			log.Fatalf("Failed to create cache dir (%s): %v", config.Net.CachePath, errMkDir)
		}
	}
	filePath := path.Join(config.Net.CachePath, list.Name)
	maxAge, errParseDuration := config.ParseDuration(config.Net.MaxAge)
	if errParseDuration != nil {
		return errors.Wrapf(errParseDuration, "Failed to parse cache max age")
	}
	expired := false
	if golib.Exists(filePath) {
		fileInfo, errStat := os.Stat(filePath)
		if errStat != nil {
			return errors.Wrapf(errStat, "Failed to stat cached file")
		}
		if config.Now().Sub(fileInfo.ModTime()) > maxAge {
			expired = true
		}
	} else {
		expired = true
	}
	if expired {
		if errDownload := download(list.URL, filePath); errDownload != nil {
			return errors.Wrapf(errDownload, "Failed to download net ban list")
		}
	}
	body, errReadFile := ioutil.ReadFile(filePath)
	if errReadFile != nil {
		return errReadFile
	}
	count, errLoadBody := load(body, list.Type)
	if errLoadBody != nil {
		return errors.Wrapf(errLoadBody, "Failed to load list")
	}
	log.WithFields(log.Fields{"count": count, "list": list.Name, "type": "steam"}).Debugf("Loaded blocklist")
	return nil
}

func download(url string, savePath string) error {
	response, errQuery := util.NewHTTPClient().Get(url)
	if errQuery != nil {
		return errQuery
	}
	outFile, errCreate := os.Create(savePath)
	if errCreate != nil {
		return errQuery
	}
	_, errCopy := io.Copy(outFile, response.Body)
	if errCopy != nil {
		return errCopy
	}
	defer func() {
		if errClose := response.Body.Close(); errClose != nil {
			log.Warnf("Failed to close block list response body: %v", errClose)
		}
	}()
	return nil
}

func load(src []byte, listType config.BanListType) (count int, err error) {
	switch listType {
	case config.CIDR:
		nets, errParseCIDR := parseCIDR(src)
		if errParseCIDR != nil {
			return 0, errParseCIDR
		}
		return addNets(nets), nil
	case config.ValveNet:
		nets, errParseValveNet := parseValveNet(src)
		if errParseValveNet != nil {
			return 0, errParseValveNet
		}
		return addNets(nets), nil
	case config.ValveSID:
		ids, errParseValveSID := parseValveSID(src)
		if errParseValveSID != nil {
			return 0, errParseValveSID
		}
		return addSIDs(ids), nil
	case config.TF2BD:
		ids, errParseBD := parseTF2BD(src)
		if errParseBD != nil {
			return 0, errParseBD
		}
		return addSIDs(ids), nil
	default:
		return 0, errors.Errorf("Unimplemented list type: %v", listType)
	}
}

func addNets(networks []*net.IPNet) int {
	count := 0
	for _, network := range networks {
		if !containsIP(network.IP) {
			networks = append(networks, network)
			count++
		}
	}
	return count
}

func addSIDs(steamIds steamid.Collection) int {
	count := 0
	for _, sid64 := range steamIds {
		if !containsSID(sid64) {
			steamids = append(steamids, sid64)
			count++
		}
	}
	return count
}

func parseCIDR(src []byte) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, line := range strings.Split(string(src), "\n") {
		if line == "" {
			continue
		}
		_, ipNet, errParseCIDR := net.ParseCIDR(line)
		if errParseCIDR != nil {
			continue
		}
		nets = append(nets, ipNet)
	}
	return nets, nil
}
