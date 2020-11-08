/*
 * Copyright © 2020 Camilo Hernández <me@camiloh.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package beekeeper

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// nodeConn is an alias for net.TCPConn.
type nodeConn net.TCPConn

// sendFunction is used to manage message sending. It gets replaced in testing.
var sendFunction = defaultSendFunction

// connectFunction is used to manage new connections. It gets replaced in testing.
var connectFunction = defaultNodeConnFunction

// defaultNodeConnFunction creates a connection with the ip. It exists to allow for testing without actually
// creating connections.
func defaultNodeConnFunction(ip string, timeout ...time.Duration) (*nodeConn, error) {
	var d net.Dialer
	if len(timeout) > 0 {
		d = net.Dialer{Timeout: timeout[0]}
	} else {
		d = net.Dialer{}
	}

	conn, err := d.Dial("tcp", setOutPortIfMissing(ip))
	if err != nil {
		return nil, err
	}

	return (*nodeConn)(conn.(*net.TCPConn)), nil
}

// newNodeConn establishes a new connection to the node using TCP.
func newNodeConn(ip string, timeout ...time.Duration) (*nodeConn, error) {
	return connectFunction(ip, timeout...)
}

// defaultSendFunction is used to send messages. It exists to allow for testing without actually sending messages.
func defaultSendFunction(c *nodeConn, m Message) error {
	m.SentAt = time.Now()
	m.From = mySettings.Name
	m.Status = mySettings.Status
	m.Token = mySettings.Config.Token

	if m.RespondOnPort == 0 {
		m.RespondOnPort = mySettings.Config.InboundPort
	}

	m.NodeInfo.OS = runtime.GOOS

	data, err := m.encode()
	if err != nil {
		return err
	}

	header := []byte(fmt.Sprintf("%d\n", len(data)))
	data = append(header, data...)

	err = c.SetWriteBuffer(len(data))
	if err != nil {
		return err
	}

	_, err = c.Write(data)
	if err != nil {
		return err
	}

	if mySettings.Config.Debug {
		log.Println("Sent:", m.summary())
	}

	return nil
}

// send fills the Message with the required metadata and sends it.
func (c *nodeConn) send(m Message) error {
	return sendFunction(c, m)
}

// getHostname uses the local network name to fetch the host system's name.
func getHostname() (name string, err error) {
	name, err = os.Hostname()
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(name, ".local", ""), nil
}

// setOutPortIfMissing adds the configured port (or default if none) to the given IP has no ports set.
func setOutPortIfMissing(ip string) string {
	if strings.Contains(ip, ":") {
		// Port already set
		return ip
	}

	port := mySettings.Config.OutboundPort
	if port == 0 {
		port = DefaultPort
	}

	return ip + ":" + strconv.Itoa(port)
}

// getLocalIP returns the primary non-loopback local address of the machine.
func getLocalIP() (ip net.IP, err error) {
	conn, err := net.Dial("udp", "1.2.3.4:80")
	if err != nil {
		return
	}

	ip = conn.LocalAddr().(*net.UDPAddr).IP
	err = conn.Close()

	return
}
