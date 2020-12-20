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
	"github.com/google/go-cmp/cmp"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAwaitTaskWithTimeoutReceived(t *testing.T) {
	s, receiveChan, _ := startPrimaryTestChannels()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		_, err := s.awaitTask("test", time.Second*10)
		if err != nil {
			t.Fail()
		}
	}()

	msg := newMessage()
	msg.Operation = OperationJobResult

	msg, err := msg.setData(Result{UUID: "test"})
	if err != nil {
		t.Fail()
	}

	receiveChan <- Request{msg, Conn{}}

	wg.Wait()
}

func TestAwaitTaskWithTimeoutTimeout(t *testing.T) {
	s, _, _ := startPrimaryTestChannels()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		_, err := s.awaitTask("test", time.Millisecond*100)
		if err != nil {
			if err == ErrTimeout {
				return
			}

			t.Fail()
		}
	}()

	wg.Wait()
}

func TestAwaitTask(t *testing.T) {
	s, receiveChan, _ := startPrimaryTestChannels()

	var wg sync.WaitGroup
	wg.Add(1)

	expect := Result{
		UUID: "test",
		Task: Task{
			UUID:      "test",
			Arguments: map[string]interface{}{"testArg1": 1, "testArg2": "testVal"},
			Returns:   map[string]interface{}{"testRet1": 1, "testRet2": "testVal"},
			Error:     "tesError",
		}}

	go func() {
		defer wg.Done()

		res, err := s.awaitTask("test")
		if err != nil {
			t.Error(err)
			return
		}

		if !cmp.Equal(res, expect) {
			t.Fail()
			return
		}
	}()

	time.Sleep(time.Millisecond * 100)

	msg := newMessage()
	msg.Operation = OperationJobResult

	msg, err := msg.setData(expect)
	if err != nil {
		t.Fail()
		return
	}

	receiveChan <- Request{msg, Conn{}}

	wg.Wait()
}

func TestAwaitTransferAndCheckAcknowledge(t *testing.T) {
	s, receiveChan, _ := startPrimaryTestChannels()

	var wg sync.WaitGroup
	wg.Add(1)

	addr := &net.TCPAddr{}

	go func() {
		defer wg.Done()

		err := s.awaitTransfer(Node{Addr: addr})
		if err != nil {
			t.Error(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 10) // Goroutine might execute last

	msg := newMessage()
	msg.Operation = OperationTransferAcknowledge
	msg.Addr = addr

	receiveChan <- Request{msg, Conn{}}

	wg.Wait()
}

func TestAwaitTransferAndCheckFailed(t *testing.T) {
	s, receiveChan, _ := startPrimaryTestChannels()

	var wg sync.WaitGroup
	wg.Add(1)

	addr := &net.TCPAddr{}

	go func() {
		defer wg.Done()

		err := s.awaitTransfer(Node{Addr: addr}, 2)
		if err == nil {
			t.Fail()
			return
		}
	}()

	time.Sleep(time.Millisecond * 10) // Goroutine might execute last

	msg := newMessage()
	msg.Operation = OperationTransferFailed
	msg.Addr = addr

	receiveChan <- Request{msg, Conn{}}

	wg.Wait()
}
