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
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Conn represents a TLS connection
type Conn struct {
	*tls.Conn
}

// dial establishes a new connection to the node using TLS over TCP.
func (s *Server) dial(ip string, timeout ...time.Duration) (*Conn, error) {
	return s.connCallback(s, ip, timeout...)
}

// defaultConnCallback creates a connection with the ip. It exists to allow for testing without actually
// creating connections.
func defaultConnCallback(s *Server, ip string, timeout ...time.Duration) (*Conn, error) {
	cert, err := tls.X509KeyPair(s.Config.TLSCertificate, s.Config.TLSPrivateKey)
	if err != nil {
		log.Fatal("Failed to parse TLS certificate")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	var d *net.Dialer
	if len(timeout) > 0 {
		d = &net.Dialer{Timeout: timeout[0]}
	} else {
		d = &net.Dialer{}
	}

	tlsConn, err := tls.DialWithDialer(d, "tcp", setOutPortIfMissing(ip, s.Config.OutboundPort), tlsConfig)
	if err != nil {
		return nil, err
	}

	go s.handle(tlsConn) // Be prepared to receive on this conn

	conn := Conn{tlsConn}
	return &conn, nil
}

// defaultSendCallback is used to sendWithConn messages. It exists to allow for testing without actually sending messages.
func defaultSendCallback(s *Server, c *Conn, m Message) error {
	m.SentAt = time.Now()
	m.Name = s.Config.Name
	m.Status = s.Status
	m.Token = s.Config.Token

	if m.RespondOnPort == 0 {
		m.RespondOnPort = s.Config.InboundPort
	}

	m.NodeInfo.OS = runtime.GOOS

	data, err := m.encode()
	if err != nil {
		return err
	}

	header := []byte(fmt.Sprintf("%d\n", len(data)))
	data = append(header, data...)

	_, err = c.Write(data)
	if err != nil {
		return err
	}

	if s.Config.Debug {
		log.Println("Sent:", m.summary())
	}

	return nil
}

// getHostname uses the local network name to fetch the host system's name.
func getHostname() (name string, err error) {
	name, err = os.Hostname()
	if err != nil {
		return "", err
	}

	return strings.Replace(name, ".local", "", -1), nil
}

// setOutPortIfMissing adds the configured port (or default if none) to the given IP has no ports set.
func setOutPortIfMissing(ip string, port int) string {
	if strings.Contains(ip, ":") {
		// Port already set
		return ip
	}
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
	_ = conn.Close()

	return
}
