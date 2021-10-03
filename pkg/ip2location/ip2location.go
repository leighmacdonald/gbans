package ip2location

import (
	"archive/zip"
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	geoDownloadURL = "https://www.ip2location.com/download/?token=%s&file=%s"

	geoDatabaseASN4     = "DBASNLITE"
	geoDatabaseASNFile4 = "IP2LOCATION-LITE-ASN.CSV"
	geoDatabaseASN6     = "DBASNLITEIPV6"
	geoDatabaseASNFile6 = "IP2LOCATION-LITE-ASN.IPV6.CSV"

	geoDatabaseLocation4     = "DB5LITECSV"
	geoDatabaseLocationFile4 = "IP2LOCATION-LITE-DB5.CSV"
	geoDatabaseLocation6     = "DB5LITECSVIPV6"
	geoDatabaseLocationFile6 = "IP2LOCATION-LITE-DB5.IPV6.CSV"

	// No ipv6 for proxy
	geoDatabaseProxy     = "PX10LITECSV"
	geoDatabaseProxyFile = "IP2PROXY-LITE-PX10.CSV"
)

type ProxyType string

const PUB ProxyType = "PUB"

type ThreatType string

const (
	ThreatUnknown           ThreatType = "-"
	ThreatSpam              ThreatType = "SPAM"
	ThreatBotnet            ThreatType = "BOTNET"
	ThreatScanner           ThreatType = "SCANNER"
	ThreatSpamBotnet        ThreatType = "SPAM/BOTNET"
	ThreatSpamScanner       ThreatType = "SPAM/SCANNER"
	ThreatSpamScannerBotnet ThreatType = "SPAM/SCANNER/BOTNET"
)

type UsageType string

const (
	UsageContentDeliveryNetwork UsageType = "CDN"
	UsageISPFixedMobile         UsageType = "ISP/MOB"
	UsageCommercial             UsageType = "COM"
	UsageISPMobile              UsageType = "MOB"
	UsageLibrary                UsageType = "LIB"
	UsageDataCenter             UsageType = "DCH"
	UsageMilitary               UsageType = "MIL"
	UsageGovernment             UsageType = "GOV"
	UsageISPFixed               UsageType = "ISP"
	UsageOrganization           UsageType = "ORG"
	UsageEducation              UsageType = "EDU"
)

// LocationRecord
// "16781312","16785407","JP","Japan","Tokyo","Tokyo","35.689506","139.691700"
// ip_from 	INT (10)† / DECIMAL (39,0)†† 	First IP address show netblock.
// ip_to 	INT (10)† / DECIMAL (39,0)†† 	Last IP address show netblock.
// country_code 	CHAR(2) 	Two-character country code based on ISO 3166.
// country_name 	VARCHAR(64) 	Country name based on ISO 3166.
// region_name 	VARCHAR(128) 	Region or state name.
// city_name 	VARCHAR(128) 	City name.
// latitude 	DOUBLE 	City latitude. Default to capital city latitude if city is unknown.
// longitude 	DOUBLE 	City longitude. Default to capital city longitude if city is unknown.
type LocationRecord struct {
	IPFrom      *net.IP
	IPTo        *net.IP
	CountryCode string
	CountryName string
	RegionName  string
	CityName    string
	LatLong     LatLong
}

// ProxyRecord
// ip_from 	INT (10)† / DECIMAL (39,0)†† 	First IP address show netblock.
// ip_to 	INT (10)† / DECIMAL (39,0)†† 	Last IP address show netblock.
// proxy_type 	VARCHAR(3) 	Type of proxy
// country_code 	CHAR(2) 	Two-character country code based on ISO 3166.
// country_name 	VARCHAR(64) 	Country name based on ISO 3166.
// region_name 	VARCHAR(128) 	Region or state name.
// city_name 	VARCHAR(128) 	City name.
// isp 	VARCHAR(256) 	Internet Service Provider or company's name.
// domain 	VARCHAR(128) 	Internet domain name associated with IP address range.
// usage_type 	VARCHAR(11) 	Usage type classification of ISP or company.
//    (COM) Commercial
//    (ORG) Organization
//    (GOV) Government
//    (MIL) Military
//    (EDU) University/College/School
//    (LIB) Library
//    (CDN) Content Delivery Network
//    (ISP) Fixed Line ISP
//    (MOB) Mobile ISP
//    (DCH) Data Center/Web Hosting/Transit
//    (SES) Search Engine Spider
//    (RSV) Reserved
// asn 	INT(10) 	Autonomous system number (ASN).
// as 	VARCHAR(256) 	Autonomous system (AS) name.
// last_seen 	INT(10) 	Proxy last seen in days.
// threat 	VARCHAR(128) 	Security threat reported.
type ProxyRecord struct {
	IPFrom      *net.IP
	IPTo        *net.IP
	ProxyType   ProxyType
	CountryCode string
	CountryName string
	RegionName  string
	CityName    string
	ISP         string
	Domain      string
	UsageType   UsageType
	ASN         int64
	AS          string
	LastSeen    time.Time
	Threat      ThreatType
}

// ASNRecord
// ip_from 	INT (10)† / DECIMAL (39,0)†† 	First IP address show netblock.
// ip_to 	INT (10)† / DECIMAL (39,0)†† 	Last IP address show netblock.
// cidr 	VARCHAR(43) 	IP address range in CIDR.
// asn 	INT(10) 	Autonomous system number (ASN).
// as 	VARCHAR(256) 	Autonomous system (AS) name.
type ASNRecord struct {
	IPFrom *net.IP
	IPTo   *net.IP
	CIDR   *net.IPNet
	ASNum  uint64
	ASName string
}

type ASNRecords []ASNRecord

func (r ASNRecords) Hosts() uint32 {
	total := uint32(0)
	for _, n := range r {
		total += util.IP2Int(*n.IPTo) - util.IP2Int(*n.IPFrom)
	}
	return total
}

type LatLong struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Location provides a container and some helper functions for location data
type Location struct {
	ISOCode string
	LatLong LatLong
	// Autonomous system number (ASN)
	ASN uint64
	// Autonomous system (AS) name
	AS string
}

// Value implements the driver.Valuer interface for our custom type
func (ll *LatLong) Value() (driver.Value, error) {
	return fmt.Sprintf("POINT(%s)", ll.String()), nil
}

// Scan implements the sql.Scanner interface for conversion to our custom type
func (ll *LatLong) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	llStrB, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to string")
	}
	llStr := string(llStrB)
	ss := strings.Split(strings.Replace(llStr, ")", "", 1), "(")
	if len(ss) != 2 {
		return errors.New("Failed to parse location")
	}
	pcs := strings.Split(ss[1], " ")
	if len(pcs) != 2 {
		return errors.New("Failed to parse location")
	}
	lon, err := strconv.ParseFloat(pcs[0], 64)
	if err != nil {
		return errors.New("Failed to parse longitude")
	}
	lat, err2 := strconv.ParseFloat(pcs[1], 64)
	if err2 != nil {
		return errors.New("Failed to parse latitude")
	}
	ll.Longitude = lon
	ll.Latitude = lat
	return nil
}

// String returns a comma separated lat long pair string
func (ll LatLong) String() string {
	return fmt.Sprintf("POINT(%f %f)", ll.Latitude, ll.Longitude)
}

// Update will fetch a new geoip database from maxmind and install it, uncompressed,
// into the configured geodb_path config file path usually defined in the configuration
// files.
func Update(outputPath string, apiKey string) error {
	type dlParam struct {
		dbName   string
		fileName string
	}
	dl := func(u dlParam) error {
		resp, err := http.Get(fmt.Sprintf(geoDownloadURL, apiKey, u.dbName))
		if err != nil {
			return errors.Wrap(err, "Failed to downloaded geoip db")
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := resp.Body.Close(); err != nil {
			log.Error("Failed to close response body for geodb download")
		}

		err2 := extractZip(b, outputPath, u.fileName)

		return err2
	}
	if apiKey == "" {
		return errors.New("invalid maxmind api key")
	}
	var exitErr error
	var wg sync.WaitGroup
	for _, u := range []dlParam{
		{dbName: geoDatabaseASN4, fileName: geoDatabaseASNFile4},
		{dbName: geoDatabaseASN6, fileName: geoDatabaseASNFile6},
		{dbName: geoDatabaseLocation4, fileName: geoDatabaseLocationFile4},
		{dbName: geoDatabaseLocation6, fileName: geoDatabaseLocationFile6},
		{dbName: geoDatabaseProxy, fileName: geoDatabaseProxyFile},
	} {
		wg.Add(1)
		req := u
		go func(params dlParam) {
			defer wg.Done()
			fi, err := os.Stat(path.Join(outputPath, params.fileName))
			if err == nil {
				age := time.Since(fi.ModTime())
				if age < time.Hour*24 {
					log.Debugf("Skipping download of: %s", params.dbName)
					return
				}
			}
			log.Infof("Downloading geodb, cache out of date: %s", params.dbName)
			if err := dl(req); err != nil {
				log.Errorf("Failed to download geo database: %s", err.Error())
			}
		}(req)
	}
	wg.Wait()
	log.Info("Update complete")
	return exitErr
}

// New opens the .mmdb file for querying and sets up the ellipsoid configuration for more accurate
// geo queries
func readASNRecords(path string, ipv6 bool) ([]ASNRecord, error) {
	var (
		records []ASNRecord
	)
	asnFile, err1 := os.Open(path)
	if err1 != nil {
		return nil, err1
	}
	reader := csv.NewReader(asnFile)
	for {
		row, err2 := reader.Read()
		if err2 == io.EOF {
			break
		}
		if err2 != nil {
			log.Fatalf("Failed to read csv row: %s", err2.Error())
		}
		ipFrom, e1 := stringInt2ip(row[0], ipv6)
		if e1 != nil {
			log.Warnf("Failed to parse ip record: %v", e1)
			continue
		}
		ipTo, e2 := stringInt2ip(row[1], ipv6)
		if e2 != nil {
			log.Warnf("Failed to parse ip record: %v", e2)
			continue
		}
		_, cidr, err2 := net.ParseCIDR(row[2])
		if err2 != nil {
			continue
		}
		asNum, err := strconv.ParseUint(row[3], 10, 64)
		if err != nil {
			continue
		}
		records = append(records, ASNRecord{IPFrom: &ipFrom, IPTo: &ipTo, CIDR: cidr, ASNum: asNum, ASName: row[4]})
	}
	return records, nil
}

func readLocationRecords(path string, ipv6 bool) ([]LocationRecord, error) {
	var records []LocationRecord
	asnFile, err1 := os.Open(path)
	if err1 != nil {
		return nil, err1
	}
	reader := csv.NewReader(asnFile)
	for {
		row, err2 := reader.Read()
		if err2 == io.EOF {
			break
		}

		ipFrom, e1 := stringInt2ip(row[0], ipv6)
		if e1 != nil {
			log.Warnf("Failed to parse ip record: %v", e1)
			continue
		}
		ipTo, e2 := stringInt2ip(row[1], ipv6)
		if e2 != nil {
			log.Warnf("Failed to parse ip record: %v", e2)
			continue
		}
		records = append(records, LocationRecord{
			IPFrom:      &ipFrom,
			IPTo:        &ipTo,
			CountryCode: row[2],
			CountryName: row[3],
			RegionName:  row[4],
			CityName:    row[5],
			LatLong: LatLong{
				util.StringToFloat64(row[6], 0),
				util.StringToFloat64(row[7], 0),
			}})
	}
	return records, nil
}

func readProxyRecords(path string) ([]ProxyRecord, error) {
	var records []ProxyRecord
	asnFile, err1 := os.Open(path)
	if err1 != nil {
		return nil, err1
	}
	reader := csv.NewReader(asnFile)
	for {
		row, err2 := reader.Read()
		if err2 == io.EOF {
			break
		}
		ipFrom, e1 := stringInt2ip(row[0], false)
		if e1 != nil {
			log.Warnf("Failed to parse ip record: %v", e1)
			continue
		}
		ipTo, e2 := stringInt2ip(row[1], false)
		if e2 != nil {
			log.Warnf("Failed to parse ip record: %v", e2)
			continue
		}
		asn := int64(0)
		var err error
		if row[10] != "-" {
			asn, err = strconv.ParseInt(row[10], 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to convert asn: %s (%s)", row[10], err)
			}
		}

		t, err := strconv.ParseInt(row[12], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to convert last_seen: %s (%s)", row[10], err)
		}
		records = append(records, ProxyRecord{
			IPFrom:      &ipFrom,
			IPTo:        &ipTo,
			ProxyType:   ProxyType(row[2]),
			CountryCode: row[3],
			CountryName: row[4],
			RegionName:  row[5],
			CityName:    row[6],
			ISP:         row[7],
			Domain:      row[8],
			UsageType:   UsageType(row[9]),
			ASN:         asn,
			AS:          row[11],
			LastSeen:    time.Unix(t, 0),
			Threat:      ThreatType(row[13]),
		})
	}
	return records, nil
}

func parseIpv6Int(s string) (net.IP, error) {
	intipv6 := big.NewInt(0)
	intipv6.SetString(s, 10)
	ip := intipv6.Bytes()
	var a net.IP = ip
	return a, nil
}

func parseIpv4Int(s string) (net.IP, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return nil, err
	}
	nn := uint32(n)
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip, nil
}

func stringInt2ip(s string, ipv6 bool) (net.IP, error) {
	if ipv6 {
		return parseIpv6Int(s)
	}
	return parseIpv4Int(s)
}

func extractZip(data []byte, dest string, filename string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, errO := f.Open()
		if errO != nil {
			return errO
		}
		defer func() {
			if errC := rc.Close(); errC != nil {
				log.Errorf("Failed to close zip: %v", errC)
			}
		}()

		p := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			if errM := os.MkdirAll(p, f.Mode()); errM != nil {
				return errM
			}
		} else {
			if errD := os.MkdirAll(filepath.Dir(p), f.Mode()); errD != nil {
				return errD
			}
			fo, errF := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if errF != nil {
				return errF
			}
			defer func() {
				if errC2 := fo.Close(); errC2 != nil {
					panic(errC2)
				}
			}()

			_, err = io.Copy(fo, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, f := range r.File {
		if f.Name == filename {
			errX := extractAndWriteFile(f)
			if errX != nil {
				return errX
			}
			break
		}
	}
	return nil
}

type BlockListData struct {
	ASN4       []ASNRecord
	ASN6       []ASNRecord
	Locations4 []LocationRecord
	Locations6 []LocationRecord
	Proxies    []ProxyRecord
}

func Read(root string) (*BlockListData, error) {
	var files [][]string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, []string{path, info.Name()})
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to build file list for import")
	}
	wg := &sync.WaitGroup{}
	var cnt int
	var data BlockListData
	var errs []error
	for _, file := range files {
		wg.Add(1)
		go func(f []string) {
			defer wg.Done()
			log.Debugf("Loading: %s", f[0])
			switch f[1] {
			case geoDatabaseASNFile4:
				records, err := readASNRecords(f[0], false)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to load %s", f[0]))
					return
				}
				data.ASN4 = records
				cnt = len(records)
			case geoDatabaseASNFile6:
				records, err := readASNRecords(f[0], true)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to load %s", f[0]))
					return
				}
				data.ASN6 = records
				cnt = len(records)
			case geoDatabaseLocationFile4:
				records, err := readLocationRecords(f[0], false)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to load %s", f[0]))
					return
				}
				data.Locations4 = records
				cnt = len(records)
			case geoDatabaseLocationFile6:
				records, err := readLocationRecords(f[0], true)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to load %s", f[0]))
					return
				}
				data.Locations6 = records
				cnt = len(records)
			case geoDatabaseProxyFile:
				records, err := readProxyRecords(f[0])
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to load %s", f[0]))
					return
				}
				data.Proxies = records
				cnt = len(records)
			}
			log.Debugf("Records: %d", cnt)
			cnt = 0
		}(file)
	}
	wg.Wait()
	if len(errs) != 0 {
		return nil, errs[0]
	}
	return &data, nil
}
