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

var receiveChan = make(chan Message)
var sendChan = make(chan Message, 500)

func startPrimaryTestChannels() (chan Message, chan Message) {
	if testWasStarted {
		return receiveChan, sendChan
	}

	testWasStarted = true

	serveCallbackFunction = func(port int, handler func(chan Message, net.Conn)) (chan Message, error) {
		return receiveChan, nil
	}

	sendFunction = func(c *nodeConn, m Message) error {
		sendChan <- m
		return nil
	}

	connectFunction = func(ip string, timeout ...time.Duration) (*nodeConn, error) {
		return &nodeConn{}, nil
	}

	WatchdogSleep = time.Millisecond * 100

	config := NewDefaultConfig()
	config.DisableConnectionWatchdog = true

	go func() {
		err := StartPrimary(config)
		if err != nil {
			panic(err)
		}
	}()

	return receiveChan, sendChan
}

func getTestWorkers() Workers {
	return Workers{
		{
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
