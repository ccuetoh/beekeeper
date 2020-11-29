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
	"time"
)

type awaitables []awaitable

type awaitable struct {
	notify    chan Message
	checkFunc func(Message) bool
}

// ErrTimeout is produced by functions called with a timeout when the allocated time is exceeded
var ErrTimeout = errors.New("time exceeded")

// ErrNodeDisconnected is produced when a node is gets disconnected while executing an operation
var ErrNodeDisconnected = errors.New("node disconnected")

// awaitTask blocks the execution until a node sends a Result with a matching taskID.
func (s *Server) awaitTask(taskId string, timeout ...time.Duration) (Result, error) {

	notifyChan := make(chan Message, 1)

	s.awaitedLock.Lock()
	s.awaited = append(s.awaited, awaitable{
		notify: notifyChan,
		checkFunc: func(msg Message) bool {
			if msg.Operation == OperationJobResult {
				res, err := decodeResult(msg.Data)
				if err != nil {
					log.Println("Error: Unable to decode task response")
					return false
				}

				if res.UUID == taskId {
					return true
				}
			}

			return false
		},
	})
	s.awaitedLock.Unlock()

	if len(timeout) > 0 {
		// Use Timer instead of using time.After. See:
		// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
		toTimer := time.NewTimer(timeout[0])
		defer toTimer.Stop()

		select {
		case msg := <-notifyChan:
			res, _ := decodeResult(msg.Data)
			return res, nil
		case <-toTimer.C:
			return Result{}, ErrTimeout
		}
	}

	msg := <-notifyChan
	res, _ := decodeResult(msg.Data)
	return res, nil
}

// awaitTransfer blocks the execution until the node sends a transfer acknowledgement or reports a transfer error.
// If an error message is received i'll be returned. An empty string means no error was raised.
func (s *Server) awaitTransfer(n Node, timeout ...time.Duration) error {
	notifyChan := make(chan Message, 1)
	disconnectChan := newDisconnectionWatchdog(s, n, 2)

	s.awaitedLock.Lock()
	s.awaited = append(s.awaited, awaitable{
		notify: notifyChan,
		checkFunc: func(msg Message) bool {
			if msg.Operation == OperationTransferFailed || msg.Operation == OperationTransferAcknowledge &&
				msg.Addr.IP.Equal(n.Addr.IP) {
				return true
			}

			return false
		},
	})
	s.awaitedLock.Unlock()

	if len(timeout) > 0 {
		// Use Timer instead of using time.After. See:
		// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
		toTimer := time.NewTimer(timeout[0])
		defer toTimer.Stop()

		select {
		case msg := <-notifyChan:
			if msg.Operation == OperationTransferAcknowledge {
				return nil
			}

			return errors.New(string(msg.Data))
		case <-toTimer.C:
			return ErrTimeout
		case <-disconnectChan:
			return ErrNodeDisconnected
		}
	}

	select {
	case msg := <-notifyChan:
		if msg.Operation == OperationTransferAcknowledge {
			return nil
		}

		return errors.New(string(msg.Data))

	case <-disconnectChan:
		return ErrNodeDisconnected
	}

}

// awaitAny blocks the execution until the node with a matching address sends any operation
func (s *Server) awaitAny(addr string, timeout ...time.Duration) (Node, error) {
	notifyChan := make(chan Message, 1)

	resolvedAddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return Node{}, err
	}

	s.awaitedLock.Lock()
	s.awaited = append(s.awaited, awaitable{
		notify: notifyChan,
		checkFunc: func(msg Message) bool {
			if msg.Addr.IP.Equal(resolvedAddr.IP) {
				return true
			}

			return false
		},
	})
	s.awaitedLock.Unlock()

	if len(timeout) > 0 {
		// Use Timer instead of using time.After. See:
		// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
		toTimer := time.NewTimer(timeout[0])
		defer toTimer.Stop()

		select {
		case <-notifyChan:
			return s.nodes.find(resolvedAddr.IP), nil
		case <-toTimer.C:
			return Node{}, ErrTimeout
		}
	}

	<-notifyChan
	return s.nodes.find(resolvedAddr.IP), nil
}

// checkAwaited compares a Message object with the awaitables list and passes it forward if matching
func (s *Server) checkAwaited(msg Message) {
	s.awaitedLock.Lock()
	defer s.awaitedLock.Unlock()

	var remaining awaitables

	for _, a := range s.awaited {
		if a.checkFunc(msg) {
			a.notify <- msg
		} else {
			remaining = append(remaining, a)
		}
	}

	s.awaited = remaining
}
