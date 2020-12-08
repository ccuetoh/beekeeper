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
	"strconv"
	"sync"
	"time"
)

// ErrTerminated is returned when a server gets terminated
var ErrTerminated = errors.New("terminated")

// Server is a node server, that holds the configuration to be used.
type Server struct {
	// Config hold the configuration data of the server.
	Config Config

	// Status represents the action the server is currently doing.
	Status Status

	// terminationChan is used to stop the server gracefully.
	terminationChan chan bool

	// nodes keeps a list of active node connections to this server.
	nodes Nodes

	// nodesLock is a RWMutex over nodes.
	nodesLock sync.RWMutex

	// queue is a chan with the incoming Requests in queue to be processed.
	queue chan Request

	// sendCallback is the callback used when sending messages to a connection.
	sendCallback func(*Server, *Conn, Message) error

	// connCallback is the callback used when creating a new connection with a node.
	connCallback func(*Server, string, ...time.Duration) (*Conn, error)

	// serverCallback is the callback used for processing the request queue.
	serverCallback func(*Server) error

	// awaited is a slice with the awaited responses.
	awaited awaitables

	// awaitedLock is a Mutex lock over awaited.
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
		queue:           make(chan Request),
	}
}

// Start serves a node and blocks.
func (s *Server) Start() error {
	log.Println("Starting server")

	err := s.serverCallback(s)
	if err != nil {
		return err
	}

	log.Printf("Listening on port %d\n", s.Config.InboundPort)

	for {
		select {
		case <-s.terminationChan:
			return ErrTerminated
		case req := <-s.queue:
			authed := req.Msg.isTokenMatching(s.Config.Token)
			if !authed {
				continue
			}

			if s.Config.Debug {
				log.Println("Received:", req.Msg.summary())
			}

			s.updateNode(req.Msg.node())
			go s.handleMessage(&req.Conn, req.Msg)
		}
	}
}

// Stop shutdowns a running server.
func (s *Server) Stop() {
	close(s.terminationChan)
}

// Connect established a TCP over TLS connection with the given address. If no node is reachable an error will be
// returned. An optional timeout argument can be provided.
func (s *Server) Connect(ip string, timeout ...time.Duration) (Node, error) {
	conn, err := s.connCallback(s, ip, timeout...)
	if err != nil {
		return Node{}, err
	}

	err = s.sendWithConn(conn, Message{Operation: OperationStatus})
	if err != nil {
		return Node{}, err
	}

	return s.awaitAny(ip, timeout...)
}

// Scan broadcasts a status Request to all IPs and waits the provided amount for a response.
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
func (s *Server) handleMessage(conn *Conn, msg Message) {
	switch msg.Operation {
	case OperationJobResult:
		jobResultCallback(s, conn, msg) // Primary

	case OperationTransferAcknowledge:
		transferStatusCallback(s, conn, msg) // Primary

	case OperationTransferFailed:
		transferStatusCallback(s, conn, msg) // Primary

	case OperationStatus:
		statusCallback(s, conn, msg) // Node

	case OperationJobTransfer:
		jobTransferCallback(s, conn, msg) // Node

	case OperationJobExecute:
		jobExecuteCallback(s, conn, msg) // Node
	}

	node := msg.node()
	node.Conn = conn

	s.updateNode(node)
	s.checkAwaited(msg)
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
func defaultServeCallback(s *Server) error {
	cer, err := tls.X509KeyPair(s.Config.TLSCertificate, s.Config.TLSPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}

	l, err := tls.Listen("tcp", ":"+strconv.Itoa(s.Config.InboundPort), tlsConfig)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("Received invalid connection:", err)
				continue
			}

			go func() {
				s.handle(conn)
			}()
		}
	}()

	return nil
}

// send sends the provided Message to the Node.
func (s *Server) send(n Node, m Message) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error while sending to node %s: %s\n", n.Name, r)
		}
	}()

	if n.Conn == nil {
		if s.Config.Debug {
			log.Printf("Creating new connection to node %s", n.Name)
		}

		var err error
		n.Conn, err = s.dial(n.Addr.IP.String())
		if err != nil {
			return err
		}
	}

	err := s.sendWithConn(n.Conn, m)
	if err != nil {
		return err
	}

	return nil
}

// sendWithConn fills the Message with the required metadata and sends it.
func (s *Server) sendWithConn(c *Conn, m Message) error {
	return s.sendCallback(s, c, m)
}
