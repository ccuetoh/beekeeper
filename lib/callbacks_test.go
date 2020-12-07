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
	"testing"
	"time"
)

func TestStatusCallback(t *testing.T) {
	sv, _, sendChan := startPrimaryTestChannels()

	msg := getTestMessage()
	msg.Operation = OperationStatus

	go sv.handleMessage(&Conn{Conn: nil}, msg)

	select {
	case response := <-sendChan:
		if response.Operation != OperationNone {
			t.Fail()
		}

		return
	case <-time.After(time.Second * 2): // The CPU info requires at least 1 second of processing
		t.Fail()
		return
	}

}

func TestJobTransferCallback_Acknowledge(t *testing.T) {
	sv, _, sendChan := startPrimaryTestChannels()

	msg := getTestMessage()
	msg.Data = []byte("test")
	msg.Operation = OperationJobTransfer

	go sv.handleMessage(&Conn{Conn: nil}, msg)

	select {
	case response := <-sendChan:
		if response.Operation != OperationTransferAcknowledge {
			t.Fail()
		}

		return
	case <-time.After(time.Second):
		t.Fail()
		return
	}

}

func TestJobTransferCallback_Failed(t *testing.T) {
	sv, _, sendChan := startPrimaryTestChannels()

	msg := getTestMessage()
	msg.Operation = OperationJobTransfer
	msg.Data = []byte{}

	go sv.handleMessage(&Conn{Conn: nil}, msg)

	select {
	case response := <-sendChan:
		if response.Operation != OperationTransferFailed {
			t.Fail()
		}

		return
	case <-time.After(time.Second):
		t.Fail()
		return
	}

}
