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
	"fmt"
	"github.com/google/go-cmp/cmp"
	"net"
	"testing"
	"time"
)

func TestDefaultHandler(t *testing.T) {
	msg := getTestMessage()
	msgChan := make(chan Message, 1)

	server, client := net.Pipe()

	data, err := msg.encode()
	if err != nil {
		t.Error(err)
		return
	}

	header := []byte(fmt.Sprintf("%d\n", len(data)))
	data = append(header, data...)

	go func() {
		_, err = server.Write(data)
		if err != nil {
			t.Error(err)
			return
		}

		err = server.Close()
		if err != nil {
			t.Error(err)
			return
		}
	}()

	defaultHandler(msgChan, client)

	select {
	case msgReceived := <-msgChan:
		msgReceived.Addr = msg.Addr // The address is set inside the handler

		if !cmp.Equal(msgReceived, msg) {
			t.Error()
			return
		}
	case <-time.After(time.Second):
		t.Fail()
		return
	}
}

func TestDefaultHandler_NoHeader(t *testing.T) {
	msg := getTestMessage()
	msgChan := make(chan Message, 1)

	server, client := net.Pipe()

	data, err := msg.encode()
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		_, err = server.Write(data)
		if err != nil {
			t.Error(err)
			return
		}

		err = server.Close()
		if err != nil {
			t.Error(err)
			return
		}
	}()

	defaultHandler(msgChan, client)

	select {
	case <-msgChan:
		t.Fail() // No Message is expected
	case <-time.After(time.Millisecond * 100):
		return
	}
}
