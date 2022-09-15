// Package msqp is a very simple client for the valve Master Server Query Protocol
// It *only* implements the list request.
// https://developer.valvesoftware.com/wiki/Master_Server_Query_Protocol
package msqp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net"
	"time"
)

const masterBrowserHost = "hl2master.steampowered.com:27011"

// Region defines a part of the world where servers are located
type Region uint8

const (
	// USEastCoast : United States - East Coast
	USEastCoast  Region = 0x00
	USWestCoast         = 0x01
	SouthAmerica        = 0x02
	Europe              = 0x03
	Asia                = 0x04
	Australia           = 0x05
	MiddleEast          = 0x06
	Africa              = 0x07
	AllRegions          = 0xFF
)

type ServerEndpoint struct {
	IP   net.IP
	Port uint16
}

func newServerEndpoint(buffer []byte) *ServerEndpoint {
	return &ServerEndpoint{
		IP:   net.IPv4(buffer[0], buffer[1], buffer[2], buffer[3]),
		Port: binary.BigEndian.Uint16(buffer[4:]),
	}
}

func (c *ServerEndpoint) String() string {
	return fmt.Sprintf("%s:%d", c.IP.String(), c.Port)
}

func List(c *net.UDPConn, regions []Region) ([]*ServerEndpoint, error) {
	const endIp = "0.0.0.0:0"
	lastIp := endIp
	filter := "\\gamedir\\tf"
	dedupeMap := map[string]*ServerEndpoint{}
	for _, region := range regions {
		firstRequest := true
		for firstRequest || lastIp != endIp {
			r, errList := sendListRequest(c, lastIp, filter, region)
			if errList != nil {
				return nil, errors.Wrap(errList, "Failed to send list request")
			}
			if len(r) == 0 {
				// Shouldn't happen?
				break
			}
			for _, result := range r {
				dedupeMap[result.String()] = result
			}
			lastIp = r[len(r)-1].String()
			firstRequest = false
			if lastIp == endIp {
				log.Println("Got EOL")
			}
		}
	}
	var results []*ServerEndpoint
	for _, v := range dedupeMap {
		results = append(results, v)
	}
	return results, nil
}

func sendListRequest(c *net.UDPConn, ipStart string, filter string, regionCode Region) ([]*ServerEndpoint, error) {
	const queryHeader byte = 0x31
	//var regionCode byte = 0xFF
	var b bytes.Buffer
	b.WriteByte(queryHeader)
	b.WriteByte(byte(regionCode))
	b.WriteString(ipStart)
	b.WriteString(filter)

	_, errWrite := c.Write(b.Bytes())
	if errWrite != nil {
		return nil, errors.Wrap(errWrite, "Failed to write udp bytes")
	}
	buffer := make([]byte, 1600)
	if errDeadLine := c.SetReadDeadline(time.Now().Add(time.Second)); errDeadLine != nil {
		return nil, errDeadLine
	}
	n, _, errRead := c.ReadFromUDP(buffer)
	if errRead != nil {
		return nil, errors.Wrap(errWrite, "Failed to read udp bytes")
	}
	const structSize int = 6
	if n%structSize > 0 {
		return nil, errors.New("Query list response has a length which is not multiple of 6")
	}
	var endpoints []*ServerEndpoint
	var replyHeader []byte = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A}
	if !bytes.Equal(replyHeader, buffer[0:structSize]) {
		return nil, errors.New("Query list response header is malformed")
	}
	res := buffer[structSize:n]
	count := len(res)
	i := 0
	for count > 0 {
		endpoint := newServerEndpoint(res[i*structSize : i*structSize+structSize])
		count -= 6
		i++
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil
}
