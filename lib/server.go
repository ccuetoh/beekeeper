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
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
)

// ErrTerminated is returned when a server gets terminated
var ErrTerminated = errors.New("terminated")

// onlineWorkers keeps a list of all the Workers that have responded to this node.
var onlineWorkers Workers
var onlineWorkersLock sync.RWMutex

// serveCallbackFunction allows for testing of the callback.
var serveCallbackFunction = defaultServeCallback

// Server is a node server, that holds the configuration to be used.
type Server struct {
	Config          Config
	terminationChan chan bool
}

// NewServer creates a Server struct using the given config or the default if none is provided.
func NewServer(configs ...Config) *Server {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = NewDefaultConfig()
	}

	return &Server{
		Config:          config,
		terminationChan: make(chan bool),
	}
}

// Start serves a node and blocks.
func (s *Server) Start() error {
	mySettings = nodeSettingsFromConfig(s.Config) // Global settings var

	msgChan, err := serveCallbackFunction(s.Config.InboundPort, defaultHandler)
	if err != nil {
		return err
	}

	for {
		select {
		case <-s.terminationChan:
			return ErrTerminated
		case msg := <-msgChan:
			authed := msg.isTokenMatching()
			if !authed {
				continue
			}

			logReceivedIfDebug(msg)

			onlineWorkers = onlineWorkers.update(msg.worker())
			go handleMessage(msg)
		}
	}
}

// Stop shutdowns a running server
func (s *Server) Stop() {
	s.terminationChan <- true
}

// handleMessage takes a Message from the node's server and runs the corresponding operation callback.
func handleMessage(msg Message) {
	switch msg.Operation {
	case OperationJobResult:
		jobResultCallback(msg) // Primary

	case OperationTransferAcknowledge:
		transferStatusCallback(msg) // Primary

	case OperationTransferFailed:
		transferStatusCallback(msg) // Primary

	case OperationStatus:
		statusCallback(msg) // Worker

	case OperationJobTransfer:
		jobTransferCallback(msg) // Worker

	case OperationJobExecute:
		jobExecuteCallback(msg) // Worker
	}

	onlineWorkers = onlineWorkers.update(msg.worker())
}

// defaultServeCallback listens for TCP connections and sends the processed output of handler to the c chan.
func defaultServeCallback(port int, handler func(chan Message, net.Conn)) (chan Message, error) {
	c := make(chan Message)

	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	go func() {
		defer l.Close()

		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}

			go func() {
				handler(c, conn)
			}()
		}
	}()

	return c, nil
}

// logReceivedIfDebug prints a Message summary if debug mode is configured.
func logReceivedIfDebug(msg Message) {
	if mySettings.Config.Debug {
		log.Println("Received:", msg.summary())
	}
}
