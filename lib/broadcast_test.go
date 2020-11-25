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
	"testing"
	"time"
)

func TestBroadcastMessage(t *testing.T) {
	sv, _, sendChan := startPrimaryTestChannels()

	msg := newMessage()
	msg.Operation = OperationStatus
	msg, err := msg.setData("test")
	if err != nil {
		t.Error(err)
		return
	}

	err = sv.broadcastMessage(msg, true)
	if err != nil {
		t.Error(err)
		return
	}

	received := 0
	for {
		select {
		case msgReceived := <-sendChan:
			if cmp.Equal(msgReceived, msg) {
				received += 1
			} else {
				t.Fail()
				return
			}

			if received == 254 { // Possible IPs
				return
			}
		case <-time.After(time.Second):
			t.Fail()
			return
		}
	}
}

func TestBroadcastOperation(t *testing.T) {
	sv, _, sendChan := startPrimaryTestChannels()

	err := sv.broadcastOperation(OperationTransferAcknowledge, true)
	if err != nil {
		t.Error(err)
		return
	}

	received := 0
	for {
		select {
		case msgReceived := <-sendChan:
			if msgReceived.Operation == OperationTransferAcknowledge {
				received += 1
			} else {
				t.Fail()
				return
			}

			if received == 254 {
				return
			}
		case <-time.After(time.Second):
			t.Fail()
			return
		}
	}
}
