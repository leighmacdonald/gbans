package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	l, err := net.Listen("tcp4", ":5111")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}

		result := strconv.Itoa(random()) + "\n"
		c.Write([]byte(result))
	}
	c.Close()
}