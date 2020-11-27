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
	"net"
	"time"
)

var testWasStarted = false

var sendChan = make(chan Message, 500)
var server *Server

func startPrimaryTestChannels() (*Server, chan Request, chan Message) {
	if testWasStarted {
		return server, server.queue, sendChan
	}

	testWasStarted = true

	config := NewDefaultConfig()
	config.DisableConnectionWatchdog = true
	WatchdogSleep = time.Millisecond * 100
	server = NewServer(config)

	server.serverCallback = func(*Server) error {
		return nil
	}

	server.sendCallback = func(c *Conn, m Message) error {
		sendChan <- m
		return nil
	}

	server.connCallback = func(_ *Server, ip string, timeout ...time.Duration) (*Conn, error) {
		return &Conn{server: server}, nil
	}

	go func() {
		err := server.Start()
		if err != nil {
			panic(err)
		}
	}()

	return server, server.queue, sendChan
}

func getTestNodes(s *Server) Nodes {
	return Nodes{
		{
			server: s,
			Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 2000, Zone: "tcp"},
			Name:   "testWorker1",
			Status: StatusIDLE,
			Info: NodeInfo{
				CPUTemp: 32,
				Usage:   10,
				OS:      "linux",
			},
		},
		{
			server: s,
			Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 2000, Zone: "tcp"},
			Name:   "testWorker2",
			Status: StatusIDLE,
			Info: NodeInfo{
				CPUTemp: 45,
				Usage:   41,
				OS:      "darwin",
			},
		},
		{
			server: s,
			Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.3"), Port: 2000, Zone: "tcp"},
			Name:   "testWorker3",
			Status: StatusIDLE,
			Info: NodeInfo{
				CPUTemp: 36,
				Usage:   1,
				OS:      "windows",
			},
		},
		{
			server: s,
			Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.4"), Port: 2000, Zone: "tcp"},
			Name:   "testWorker4",
			Status: StatusIDLE,
			Info: NodeInfo{
				CPUTemp: 36,
				Usage:   1,
				OS:      "windows",
			},
		},
	}
}

func getTestMessage() Message {
	return Message{
		SentAt:        time.Now(),
		From:          "TEST_HOST",
		Operation:     OperationNone,
		Data:          []byte("TEST_DATA"),
		Token:         "TEST_TOKEN",
		Addr:          &net.TCPAddr{Port: 2000, IP: net.ParseIP("192.168.1.1"), Zone: "tcp"},
		RespondOnPort: 2000,
		Status:        StatusIDLE,
		NodeInfo:      NodeInfo{CPUTemp: 42.2, Usage: 5.1, OS: "linux"},
	}
}
