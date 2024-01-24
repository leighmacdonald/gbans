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
	"log"
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

	"github.com/leighmacdonald/gbans/pkg/util"
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

	// No ipv6 for proxy.
	geoDatabaseProxy     = "PX10LITECSV"
	geoDatabaseProxyFile = "IP2PROXY-LITE-PX10.CSV"
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
		dbName   string
		fileName string
	}

	downloadDatabase := func(params dlParam) error {
		client := &http.Client{
			Timeout: time.Minute * 5,
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(geoDownloadURL, apiKey, params.dbName), nil)
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

		return extractZip(body, outputPath, params.fileName)
	}

	if apiKey == "" {
		return ErrAPIKey
	}

	var (
		exitErr   error
		waitGroup sync.WaitGroup
	)

	for _, param := range []dlParam{
		{dbName: geoDatabaseASN4, fileName: geoDatabaseASNFile4},
		{dbName: geoDatabaseASN6, fileName: geoDatabaseASNFile6},
		{dbName: geoDatabaseLocation4, fileName: geoDatabaseLocationFile4},
		{dbName: geoDatabaseLocation6, fileName: geoDatabaseLocationFile6},
		{dbName: geoDatabaseProxy, fileName: geoDatabaseProxyFile},
	} {
		waitGroup.Add(1)

		go func(param dlParam) {
			defer waitGroup.Done()

			fileInfo, errStat := os.Stat(path.Join(outputPath, param.fileName))
			if errStat == nil {
				age := time.Since(fileInfo.ModTime())
				if age < time.Hour*24 {
					return
				}
			}

			if errDownload := downloadDatabase(param); errDownload != nil {
				log.Printf("Failed to download geo database: %v", errDownload)
			}
		}(param)
	}

	waitGroup.Wait()

	return exitErr
}

// New opens the .mmdb file for querying and sets up the ellipsoid configuration for more accurate
// geo queries.
func readASNRecords(path string, ipv6 bool) ([]ASNRecord, error) {
	var records []ASNRecord

	asnFile, errOpen := os.Open(path)
	if errOpen != nil {
		return nil, errors.Join(errOpen, ErrOpenFile)
	}

	reader := csv.NewReader(asnFile)

	for {
		recordLine, errReadLine := reader.Read()
		if errors.Is(errReadLine, io.EOF) {
			break
		}

		if errReadLine != nil {
			return nil, errors.Join(errReadLine, ErrCSVRow)
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], ipv6)
		if errParseFromIP != nil {
			return nil, errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], ipv6)
		if errParseToIP != nil {
			return nil, errors.Join(errParseToIP, ErrParseIP)
		}

		_, network, errParseCIDR := net.ParseCIDR(recordLine[2])
		if errParseCIDR != nil {
			return nil, errors.Join(errParseCIDR, ErrParseCIDR)
		}

		if recordLine[3] == "-" {
			continue
		}

		asNum, errParseASNum := strconv.ParseUint(recordLine[3], 10, 64)
		if errParseASNum != nil {
			return nil, errors.Join(errParseCIDR, ErrParseASN)
		}

		records = append(records, ASNRecord{IPFrom: &ipFrom, IPTo: &ipTo, CIDR: network, ASNum: asNum, ASName: recordLine[4]})
	}

	return records, nil
}

func readLocationRecords(path string, ipv6 bool) ([]LocationRecord, error) {
	var records []LocationRecord

	asnFile, errOpen := os.Open(path)
	if errOpen != nil {
		return nil, errors.Join(errOpen, ErrOpenFile)
	}

	reader := csv.NewReader(asnFile)

	for {
		recordLine, errReadLine := reader.Read()
		if errors.Is(errReadLine, io.EOF) {
			break
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], ipv6)
		if errParseFromIP != nil {
			return nil, errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], ipv6)
		if errParseToIP != nil {
			return nil, errors.Join(errParseToIP, ErrParseIP)
		}

		records = append(records, LocationRecord{
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
		})
	}

	return records, nil
}

func readProxyRecords(path string) ([]ProxyRecord, error) {
	var records []ProxyRecord

	asnFile, errOpen := os.Open(path)
	if errOpen != nil {
		return nil, errors.Join(errOpen, ErrOpenFile)
	}

	reader := csv.NewReader(asnFile)

	for {
		recordLine, errReadRecordLine := reader.Read()
		if errors.Is(errReadRecordLine, io.EOF) {
			break
		}

		ipFrom, errParseFromIP := stringInt2ip(recordLine[0], false)
		if errParseFromIP != nil {
			return nil, errors.Join(errParseFromIP, ErrParseIP)
		}

		ipTo, errParseToIP := stringInt2ip(recordLine[1], false)
		if errParseToIP != nil {
			return nil, errors.Join(errParseToIP, ErrParseIP)
		}

		asn := int64(0)

		if recordLine[10] != "-" {
			parsedAsn, errParseASN := strconv.ParseInt(recordLine[10], 10, 64)
			if errParseASN != nil {
				return nil, errors.Join(errParseASN, fmt.Errorf("failed to convert asn: %s (%w)", recordLine[10], errParseASN))
			}

			asn = parsedAsn
		}

		lastSeen, errParseLastSeen := strconv.ParseInt(recordLine[12], 10, 64)
		if errParseLastSeen != nil {
			return nil, errors.Join(errParseLastSeen, fmt.Errorf("failed to convert last_seen: %s (%w)", recordLine[10], errParseLastSeen))
		}

		records = append(records, ProxyRecord{
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
		})
	}

	return records, nil
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

func extractZip(data []byte, dest string, filename string) error {
	zipReader, errNewReader := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if errNewReader != nil {
		return errors.Join(errNewReader, ErrZipReader)
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(zipFile *zip.File) error {
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

			_, errNewReader = io.Copy(outFile, readCloser) //nolint:gosec
			if errNewReader != nil {
				return errors.Join(errNewReader, ErrCopyDest)
			}
		}

		return nil
	}

	for _, readerFile := range zipReader.File {
		if readerFile.Name == filename {
			errExtractFile := extractAndWriteFile(readerFile)
			if errExtractFile != nil {
				return errExtractFile
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
	var (
		files     [][]string
		waitGroup = &sync.WaitGroup{}
		data      BlockListData
		errs      []error
	)

	errWalkPath := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, []string{path, info.Name()})

		return nil
	})

	if errWalkPath != nil {
		return nil, errors.Join(errWalkPath, ErrFileList)
	}

	for _, file := range files {
		waitGroup.Add(1)

		go func(filePaths []string) {
			defer waitGroup.Done()

			switch filePaths[1] {
			case geoDatabaseASNFile4:
				records, errReadASN := readASNRecords(filePaths[0], false)
				if errReadASN != nil {
					errs = append(errs, errors.Join(errReadASN, fmt.Errorf("%w: %s", ErrLoad, filePaths[0])))

					return
				}

				data.ASN4 = records
			case geoDatabaseASNFile6:
				records, errReadASN := readASNRecords(filePaths[0], true)
				if errReadASN != nil {
					errs = append(errs, errors.Join(errReadASN, fmt.Errorf("%w: %s", ErrLoad, filePaths[0])))

					return
				}

				data.ASN6 = records
			case geoDatabaseLocationFile4:
				records, errReadLocation := readLocationRecords(filePaths[0], false)
				if errReadLocation != nil {
					errs = append(errs, errors.Join(errReadLocation, fmt.Errorf("%w: %s", ErrLoad, filePaths[0])))

					return
				}

				data.Locations4 = records
			case geoDatabaseLocationFile6:
				records, errReadLocation := readLocationRecords(filePaths[0], true)
				if errReadLocation != nil {
					errs = append(errs, errors.Join(errReadLocation, fmt.Errorf("%w: %s", ErrLoad, filePaths[0])))

					return
				}

				data.Locations6 = records
			case geoDatabaseProxyFile:
				records, errReadProxy := readProxyRecords(filePaths[0])
				if errReadProxy != nil {
					errs = append(errs, errors.Join(errReadProxy, fmt.Errorf("%w: %s", ErrLoad, filePaths[0])))

					return
				}

				data.Proxies = records
			}
		}(file)
	}

	waitGroup.Wait()

	if len(errs) != 0 {
		return nil, errs[0]
	}

	return &data, nil
}
