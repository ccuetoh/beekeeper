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
	"strconv"
	"strings"
	"sync"
	"time"
)

// broadcastMessage sends the Message to all IPs in the local subnetwork.
func (s *Server) broadcastMessage(msg Message, await bool) error {
	return broadcastCallback(s, msg, await)
}

// broadcastOperation sends a Message containing only the op operation to all IPs in the local subnetwork.
func (s *Server) broadcastOperation(op Operation, await bool) error {
	return broadcastCallback(s, Message{Operation: op, Token: s.Config.Token}, await)
}

// broadcastCallback is the callback for the broadcast functions.
func broadcastCallback(s *Server, msg Message, await bool) error {
	myIP, err := getLocalIP()
	if err != nil {
		return err
	}

	ipComponents := strings.Split(myIP.String(), ".")
	localNetwork := strings.Join(ipComponents[:len(ipComponents)-1], ".") + "." // 192.168.0.

	myIPEnding, _ := strconv.Atoi(ipComponents[len(ipComponents)-1])

	var wg sync.WaitGroup

	for x := 1; x <= 255; x++ {
		if myIPEnding == x {
			continue
		}

		x := x
		if await {
			wg.Add(1)
		}

		go func() {
			if await {
				defer wg.Done()
			}

			ip := localNetwork + strconv.Itoa(x)

			conn, err := s.dial(ip, time.Second)
			if err != nil {
				// log.Printf("Unable to create connection while broadcasting to %s: %s\n", ip, err.Error())
				return
			}

			err = conn.send(msg)
			if err != nil {
				// log.Println("Error: Unable to send operation to node while broadcasting to", ip)
				return
			}
		}()
	}

	if await {
		wg.Wait()
	}

	return nil
}
