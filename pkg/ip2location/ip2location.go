// Package ip2location implements downloading and parsing of ip2location databases.
package ip2location

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
)

const downloadURL = "https://www.ip2location.com/download/?token=%s&file=%s"

type DatabaseName string

const (
	GeoDatabaseASN4      DatabaseName = "DBASNLITE"
	GeoDatabaseASN6      DatabaseName = "DBASNLITEIPV6"
	GeoDatabaseLocation4 DatabaseName = "DB5LITECSV"
	GeoDatabaseLocation6 DatabaseName = "DB5LITECSVIPV6"

	// No ipv6 for proxy.
	GeoDatabaseProxy DatabaseName = "PX10LITECSV"
)

type DatabaseFile string

const (
	GeoDatabaseASNFile4      DatabaseFile = "IP2LOCATION-LITE-ASN.CSV"
	GeoDatabaseASNFile6      DatabaseFile = "IP2LOCATION-LITE-ASN.IPV6.CSV"
	GeoDatabaseLocationFile4 DatabaseFile = "IP2LOCATION-LITE-DB5.CSV"
	GeoDatabaseLocationFile6 DatabaseFile = "IP2LOCATION-LITE-DB5.IPV6.CSV"
	GeoDatabaseProxyFile     DatabaseFile = "IP2PROXY-LITE-PX10.CSV"
)

var (
	ErrCSVRow        = errors.New("failed to read asn csv row")
	ErrOpenFile      = errors.New("failed to open asn file for reading")
	ErrParseASN      = errors.New("failed to parse asn record")
	ErrParseIP       = errors.New("failed to parse ip record")
	ErrParseCIDR     = errors.New("failed to parse cidr record")
	ErrZipReader     = errors.New("failed to create new zip reader")
	ErrDir           = errors.New("failed to make destination directory")
	ErrOpenDest      = errors.New("failed to open output file")
	ErrCopyDest      = errors.New("failed to copy content to output file")
	ErrInsecureZip   = errors.New("insecure zip extraction detected")
	ErrFileList      = errors.New("failed to build file list for import")
	ErrLoad          = errors.New("failed to load dataset")
	ErrConvertString = errors.New("failed to convert value to string")
	ErrParseLocation = errors.New("failed to parse location")
	ErrAPIKey        = errors.New("invalid maxmind api key")
)

type ProxyType string

const PUB ProxyType = "PUB"

type ThreatType string

// nolint
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

// nolint
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
//
//	(COM) Commercial
//	(ORG) Organization
//	(GOV) Government
//	(MIL) Military
//	(EDU) University/College/School
//	(LIB) Library
//	(CDN) Content Delivery Network
//	(ISP) Fixed Line ISP
//	(MOB) Mobile ISP
//	(DCH) Data Center/Web Hosting/Transit
//	(SES) Search Engine Spider
//	(RSV) Reserved
//
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

// Location provides a container and some helper functions for location data.
type Location struct {
	ISOCode string
	LatLong LatLong
	// Autonomous system number (ASN)
	ASN uint64
	// Autonomous system (AS) name
	AS string
}

// Value implements the driver.Valuer interface for our custom type.
func (ll *LatLong) Value() (driver.Value, error) {
	return fmt.Sprintf("POINT(%s)", ll.String()), nil
}

// Scan implements the sql.Scanner interface for conversion to our custom type.
func (ll *LatLong) Scan(value any) error {
	// Should be more strictly to check this type.
	llStrB, ok := value.([]byte)
	if !ok {
		return ErrConvertString
	}

	llStr := string(llStrB)

	ss := strings.Split(strings.Replace(llStr, ")", "", 1), "(")
	if len(ss) != 2 {
		return ErrParseLocation
	}

	pieces := strings.Split(ss[1], " ")
	if len(pieces) != 2 {
		return ErrParseLocation
	}

	lon, errParseLon := strconv.ParseFloat(pieces[0], 64)
	if errParseLon != nil {
		return ErrParseLocation
	}

	lat, errParseLat := strconv.ParseFloat(pieces[1], 64)
	if errParseLat != nil {
		return ErrParseLocation
	}

	ll.Longitude = lon
	ll.Latitude = lat

	return nil
}

// String returns a comma separated lat long pair string.
func (ll *LatLong) String() string {
	return fmt.Sprintf("POINT(%f %f)", ll.Latitude, ll.Longitude)
}

var (
	ErrCreateRequest = errors.New("failed to create request")
	ErrDownload      = errors.New("failed to downloaded geoip db")
	ErrResponse      = errors.New("failed to read response body")
	ErrClose         = errors.New("failed to close response body")
)

// Update will fetch a new geoip database from maxmind and install it, uncompressed,
// into the configured geodb_path defined in the configuration file.
func Update(ctx context.Context, outputPath string, apiKey string) error {
	type dlParam struct {
		dbName   DatabaseName
		fileName DatabaseFile
	}

	downloadDatabase := func(params dlParam) error {
		client := &http.Client{
			Timeout: time.Minute * 5,
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(downloadURL, apiKey, params.dbName), nil)
		if reqErr != nil {
			return errors.Join(reqErr, ErrCreateRequest)
		}

		resp, errGet := client.Do(req)
		if errGet != nil {
			return errors.Join(errGet, ErrDownload)
		}

		body, errReadAll := io.ReadAll(resp.Body)
		if errReadAll != nil {
			return errors.Join(errReadAll, ErrResponse)
		}

		if errCloseBody := resp.Body.Close(); errCloseBody != nil {
			return errors.Join(errCloseBody, ErrClose)
		}

		return extractZip(body, outputPath, string(params.fileName))
	}

	if apiKey == "" {
		return ErrAPIKey
	}

	var (
		exitErr   error
		waitGroup sync.WaitGroup
	)

	for _, param := range []dlParam{
		{dbName: GeoDatabaseASN4, fileName: GeoDatabaseASNFile4},
		{dbName: GeoDatabaseASN6, fileName: GeoDatabaseASNFile6},
		{dbName: GeoDatabaseLocation4, fileName: GeoDatabaseLocationFile4},
		{dbName: GeoDatabaseLocation6, fileName: GeoDatabaseLocationFile6},
		{dbName: GeoDatabaseProxy, fileName: GeoDatabaseProxyFile},
	} {
		waitGroup.Add(1)

		go func(param dlParam) {
			defer waitGroup.Done()

			fileInfo, errStat := os.Stat(path.Join(outputPath, string(param.fileName)))
			if errStat == nil {
				age := time.Since(fileInfo.ModTime())
				if age < time.Hour*24 {
					return
				}
			}

			slog.Debug("Downloading ip2location records", slog.String("db", string(param.dbName)))

			if errDownload := downloadDatabase(param); errDownload != nil {
				slog.Error("Failed to download geo database", log.ErrAttr(errDownload))
			}
		}(param)
	}

	waitGroup.Wait()

	return exitErr
}

type ASNLoader func(ctx context.Context, truncate bool, records []ASNRecord) error

func ReadASNRecords(ctx context.Context, path string, ipv6 bool, onRecords ASNLoader) error {
	asnFile, errOpen := os.Open(path)
	if errOpen != nil {
		return errors.Join(errOpen, ErrOpenFile)
	}

	defer asnFile.Close()

	var (
		reader  = csv.NewReader(asnFile)
		records []ASNRecord
		first   = true
		total   int64
		started = time.Now()
	)

	defer func() {
		slog.Info("Import asn records complete", slog.Int64("total", total), slog.Duration("duration", time.Since(started)))
	}()

	for {
		recordLine, errReadLine := reader.Read()
		if errors.Is(errReadLine, io.EOF) {
			if len(records) == 0 {
				return nil
			}

			return onRecords(ctx, first, records)
		}

		if errReadLine != nil {
			return errors.Join(errReadLine, ErrCSVRow)
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], ipv6)
		if errParseFromIP != nil {
			return errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], ipv6)
		if errParseToIP != nil {
			return errors.Join(errParseToIP, ErrParseIP)
		}

		_, network, errParseCIDR := net.ParseCIDR(recordLine[2])
		if errParseCIDR != nil {
			if recordLine[2] == "-" {
				continue
			}

			return errors.Join(errParseCIDR, ErrParseCIDR)
		}

		if recordLine[3] == "-" {
			continue
		}

		asNum, errParseASNum := strconv.ParseUint(recordLine[3], 10, 64)
		if errParseASNum != nil {
			return errors.Join(errParseCIDR, ErrParseASN)
		}

		records = append(records, ASNRecord{IPFrom: &ipFrom, IPTo: &ipTo, CIDR: network, ASNum: asNum, ASName: recordLine[4]})

		total++

		if len(records) == 10000 {
			if err := onRecords(ctx, first, records); err != nil {
				return err
			}

			slog.Debug("Imported asn records", slog.Int64("total", total))

			records = nil
			first = false
		}
	}
}

type LocationLoader func(ctx context.Context, truncate bool, records []LocationRecord) error

func ReadLocationRecords(ctx context.Context, path string, ipv6 bool, onRecords LocationLoader) error {
	locationFile, errOpen := os.Open(path)
	if errOpen != nil {
		return errors.Join(errOpen, ErrOpenFile)
	}

	defer locationFile.Close()

	var (
		reader  = csv.NewReader(locationFile)
		records []LocationRecord
		first   = true
		total   int64
		started = time.Now()
	)

	defer func() {
		slog.Info("Import location records complete", slog.Int64("total", total), slog.Duration("duration", time.Since(started)))
	}()

	for {
		recordLine, errReadLine := reader.Read()
		if errors.Is(errReadLine, io.EOF) {
			return onRecords(ctx, first, records)
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], ipv6)
		if errParseFromIP != nil {
			return errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], ipv6)
		if errParseToIP != nil {
			return errors.Join(errParseToIP, ErrParseIP)
		}

		record := LocationRecord{
			IPFrom:      &ipFrom,
			IPTo:        &ipTo,
			CountryCode: recordLine[2],
			CountryName: recordLine[3],
			RegionName:  recordLine[4],
			CityName:    recordLine[5],
			LatLong: LatLong{
				util.StringToFloat64(recordLine[6], 0),
				util.StringToFloat64(recordLine[7], 0),
			},
		}

		total++

		records = append(records, record)
		if len(records) == 10000 {
			if err := onRecords(ctx, first, records); err != nil {
				return err
			}

			slog.Debug("Imported location records", slog.Int64("total", total))

			first = false
			records = nil
		}
	}
}

type ProxyLoader func(ctx context.Context, truncate bool, records []ProxyRecord) error

func ReadProxyRecords(ctx context.Context, path string, onRecords ProxyLoader) error {
	asnFile, errOpen := os.Open(path)
	if errOpen != nil {
		return errors.Join(errOpen, ErrOpenFile)
	}

	defer asnFile.Close()

	var (
		reader  = csv.NewReader(asnFile)
		records []ProxyRecord
		first   = true
		total   int64
		started = time.Now()
	)

	defer func() {
		slog.Info("Import proxy records complete", slog.Int64("total", total), slog.Duration("duration", time.Since(started)))
	}()

	for {
		recordLine, errReadLine := reader.Read()
		if errors.Is(errReadLine, io.EOF) {
			if len(records) == 0 {
				return nil
			}

			return onRecords(ctx, first, records)
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], false)
		if errParseFromIP != nil {
			return errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], false)
		if errParseToIP != nil {
			return errors.Join(errParseToIP, ErrParseIP)
		}

		asn := int64(0)

		if recordLine[10] != "-" {
			parsedAsn, errParseASN := strconv.ParseInt(recordLine[10], 10, 64)
			if errParseASN != nil {
				return errors.Join(errParseASN, fmt.Errorf("failed to convert asn: %s (%w)", recordLine[10], errParseASN))
			}

			asn = parsedAsn
		}

		lastSeen, errParseLastSeen := strconv.ParseInt(recordLine[12], 10, 64)
		if errParseLastSeen != nil {
			return errors.Join(errParseLastSeen, fmt.Errorf("failed to convert last_seen: %s (%w)", recordLine[10], errParseLastSeen))
		}

		record := ProxyRecord{
			IPFrom:      &ipFrom,
			IPTo:        &ipTo,
			ProxyType:   ProxyType(recordLine[2]),
			CountryCode: recordLine[3],
			CountryName: recordLine[4],
			RegionName:  recordLine[5],
			CityName:    recordLine[6],
			ISP:         recordLine[7],
			Domain:      recordLine[8],
			UsageType:   UsageType(recordLine[9]),
			ASN:         asn,
			AS:          recordLine[11],
			LastSeen:    time.Unix(lastSeen, 0),
			Threat:      ThreatType(recordLine[13]),
		}

		records = append(records, record)

		total++

		if len(records) == 10000 {
			if err := onRecords(ctx, first, records); err != nil {
				return err
			}

			slog.Debug("Imported proxy records", slog.Int64("total", total))

			first = false
			records = nil
		}
	}
}

func parseIpv6Int(s string) (net.IP, error) {
	intIPv6 := big.NewInt(0)
	intIPv6.SetString(s, 10)
	ip := intIPv6.Bytes()

	var a net.IP = ip

	return a, nil
}

func parseIpv4Int(s string) (net.IP, error) {
	n, errParseInt := strconv.ParseUint(s, 10, 32)
	if errParseInt != nil {
		return nil, errors.Join(errParseInt, ErrParseIP)
	}

	nn := uint32(n)
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)

	return ip, nil
}

func stringInt2ip(ipString string, ipv6 bool) (net.IP, error) {
	if ipv6 {
		return parseIpv6Int(ipString)
	}

	return parseIpv4Int(ipString)
}

func extractAndWriteFile(zipFile *zip.File, dest string) error {
	readCloser, errOpen := zipFile.Open()
	if errOpen != nil {
		return errors.Join(errOpen, ErrOpenFile)
	}

	defer func() {
		_ = readCloser.Close()
	}()

	filePath := filepath.Join(dest, zipFile.Name) //nolint:gosec
	if strings.Contains(filePath, "..") {
		return ErrInsecureZip
	}

	if zipFile.FileInfo().IsDir() {
		if errM := os.MkdirAll(filePath, zipFile.Mode()); errM != nil {
			return errors.Join(errM, ErrDir)
		}
	} else {
		if errMkDir := os.MkdirAll(filepath.Dir(filePath), zipFile.Mode()); errMkDir != nil {
			return errors.Join(errMkDir, ErrDir)
		}

		outFile, errOpenFile := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
		if errOpenFile != nil {
			return errors.Join(errOpen, ErrOpenDest)
		}

		defer func() { _ = outFile.Close() }()

		_, errCopy := io.Copy(outFile, readCloser) //nolint:gosec
		if errCopy != nil {
			return errors.Join(errCopy, ErrCopyDest)
		}
	}

	return nil
}

func extractZip(data []byte, dest string, filename string) error {
	zipReader, errNewReader := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if errNewReader != nil {
		return errors.Join(errNewReader, ErrZipReader)
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods

	for _, readerFile := range zipReader.File {
		if readerFile.Name == filename {
			errExtractFile := extractAndWriteFile(readerFile, dest)
			if errExtractFile != nil {
				return errExtractFile
			}

			break
		}
	}

	return nil
}
