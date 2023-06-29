// Package msqp is a very simple client for the valve Master Server Query Protocol
// It *only* implements the list request.
// https://developer.valvesoftware.com/wiki/Master_Server_Query_Protocol
package msqp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
)

const masterBrowserHost = "hl2master.steampowered.com:27011"

// Region defines a part of the world where servers are located.
type Region uint8

const (
	// USEastCoast : United States - East Coast.
	USEastCoast  Region = 0x00
	USWestCoast  Region = 0x01
	SouthAmerica Region = 0x02
	Europe       Region = 0x03
	Asia         Region = 0x04
	Australia    Region = 0x05
	MiddleEast   Region = 0x06
	Africa       Region = 0x07
	AllRegions   Region = 0xFF
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
			// if lastIp == endIp {
			//	fmt.Println("Got EOL")
			//}
		}
	}
	var results []*ServerEndpoint
	for _, v := range dedupeMap {
		results = append(results, v)
	}

	return results, nil
}

func sendListRequest(conn *net.UDPConn, ipStart string, filter string, regionCode Region) ([]*ServerEndpoint, error) {
	const queryHeader byte = 0x31
	var buf bytes.Buffer
	buf.WriteByte(queryHeader)
	buf.WriteByte(byte(regionCode))
	buf.WriteString(ipStart)
	buf.WriteString(filter)
	_, errWrite := conn.Write(buf.Bytes())
	if errWrite != nil {
		return nil, errors.Wrap(errWrite, "Failed to write udp bytes")
	}
	buffer := make([]byte, 1600)
	if errDeadLine := conn.SetReadDeadline(time.Now().Add(time.Second)); errDeadLine != nil {
		return nil, errDeadLine
	}
	readCount, _, errRead := conn.ReadFromUDP(buffer)
	if errRead != nil {
		return nil, errors.Wrap(errWrite, "Failed to read udp bytes")
	}
	const structSize int = 6
	if readCount%structSize > 0 {
		return nil, errors.New("Query list response has a length which is not multiple of 6")
	}
	var endpoints []*ServerEndpoint
	replyHeader := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A}
	if !bytes.Equal(replyHeader, buffer[0:structSize]) {
		return nil, errors.New("Query list response header is malformed")
	}
	res := buffer[structSize:readCount]
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
