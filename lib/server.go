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
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// ErrTerminated is returned when a server gets terminated
var ErrTerminated = errors.New("terminated")

// Server is a node server, that holds the configuration to be used.
type Server struct {
	// Public
	Config Config
	Status Status

	// Termination
	terminationChan chan bool

	// Active nodes
	nodes     Nodes
	nodesLock sync.RWMutex

	// Callbacks
	sendCallback   func(*Conn, Message) error
	connCallback   func(*Server, string, ...time.Duration) (*Conn, error)
	serverCallback func(Config, func(chan Message, net.Conn)) (chan Message, error)

	// Awaited
	awaited     awaitables
	awaitedLock sync.Mutex
}

// NewServer creates a Server struct using the given config or the default if none is provided.
func NewServer(configs ...Config) *Server {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = NewDefaultConfig()
	}

	if config.TLSCertificate == nil || config.TLSPrivateKey == nil {
		var err error
		config.TLSCertificate, config.TLSPrivateKey, err = getTLSCache()
		if err != nil {
			log.Println("Creating TLS certificates. This can take a while but is only done once")

			config.TLSCertificate, config.TLSPrivateKey, err = newSelfSignedCert()
			if err != nil {
				log.Panicln("Unable to create TLS certificate")
			}

			err = cacheTLS(config.TLSCertificate, config.TLSPrivateKey)
			if err != nil {
				log.Println("Unable to cache TLS certificate:", err.Error())
			}
		}
	}

	return &Server{
		Config:          config,
		terminationChan: make(chan bool),
		connCallback:    defaultConnCallback,
		sendCallback:    defaultSendCallback,
		serverCallback:  defaultServeCallback,
	}
}

// Start serves a node and blocks.
func (s *Server) Start() error {
	log.Println("Starting server")

	msgChan, err := s.serverCallback(s.Config, defaultHandler)
	if err != nil {
		return err
	}

	log.Printf("Listening on port %d\n", s.Config.InboundPort)

	for {
		select {
		case <-s.terminationChan:
			return ErrTerminated
		case msg := <-msgChan:
			authed := msg.isTokenMatching(s.Config.Token)
			if !authed {
				continue
			}

			if s.Config.Debug {
				log.Println("Received:", msg.summary())
			}

			s.updateNode(msg.node())
			go s.handleMessage(msg)
		}
	}
}

// Stop shutdowns a running server
func (s *Server) Stop() {
	s.terminationChan <- true
}

// Scan broadcasts a status request to all IPs and waits the provided amount for a response.
func (s *Server) Scan(waitTime time.Duration) (Nodes, error) {
	err := s.broadcastOperation(OperationStatus, false)
	if err != nil {
		return nil, err
	}

	time.Sleep(waitTime)

	s.nodesLock.RLock()
	defer s.nodesLock.RUnlock()

	return s.nodes, nil
}

// handleMessage takes a Message from the node's server and runs the corresponding operation callback.
func (s *Server) handleMessage(msg Message) {
	switch msg.Operation {
	case OperationJobResult:
		jobResultCallback(s, msg) // Primary

	case OperationTransferAcknowledge:
		transferStatusCallback(s, msg) // Primary

	case OperationTransferFailed:
		transferStatusCallback(s, msg) // Primary

	case OperationStatus:
		statusCallback(s, msg) // Node

	case OperationJobTransfer:
		jobTransferCallback(s, msg) // Node

	case OperationJobExecute:
		jobExecuteCallback(s, msg) // Node
	}

	s.updateNode(msg.node())
}

// isOnline searches the node in the server's node slice
func (s *Server) isOnline(n Node) bool {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	for _, node := range s.nodes {
		if n.Equals(node) {
			return true
		}
	}

	return false
}

// defaultServeCallback listens for TCP connections and sends the processed output of handler to the c chan.
func defaultServeCallback(config Config, handler func(chan Message, net.Conn)) (chan Message, error) {
	c := make(chan Message)

	cer, err := tls.X509KeyPair(config.TLSCertificate, config.TLSPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}

	l, err := tls.Listen("tcp", ":"+strconv.Itoa(config.InboundPort), tlsConfig)
	if err != nil {
		return nil, err
	}

	go func() {
		defer l.Close()

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("Received invalid connection:", err)
				continue
			}

			go func() {
				handler(c, conn)
			}()
		}
	}()

	return c, nil
}
