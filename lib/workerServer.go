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

import "log"

// StartPrimary runs a server for a worker node and blocks. An optional Config can be provided. If none is passed,
// a default configuration is used.
func StartWorker(configs ...Config) error {
	c := make(chan Message)

	var config Config
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = NewDefaultConfig()
	}

	mySettings = nodeSettingsFromConfig(config)

	err := serveCallback(c, config.InboundPort, defaultHandler)
	if err != nil {
		return err
	}

	for {
		msg := <-c

		authed := msg.isTokenMatching()
		if !authed {
			if config.Debug {
				log.Println("Received no-auth:", msg.summary())
			}

			continue
		}

		logReceivedIfDebug(msg)

		go workerHandleMessage(msg)
	}
}

// workerHandleMessage takes a Message from the worker node's server and runs the corresponding operation callback.
func workerHandleMessage(msg Message) {
	switch msg.Operation {
	case OperationStatus:
		workerStatusCallback(msg)

	case OperationJobTransfer:
		workerJobTransferCallback(msg)

	case OperationJobExecute:
		workerJobExecuteCallback(msg)
	}
}
