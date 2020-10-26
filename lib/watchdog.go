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
	"log"
	"time"
)

// startConnectionWatchdog will periodically clear the online workers list and broadcastOperation a new status request to
// refill it.
func startConnectionWatchdog() {
	for {
		time.Sleep(DefaultWatchdogSleep)

		onlineWorkers = Workers{}
		err := broadcastOperation(OperationStatus, false)
		if err != nil {
			log.Println("Unable to broadcastOperation as watchdog:", err.Error())
		}
	}
}

// newDisconnectionWatchdog checks every DefaultWatchdogSleep seconds if a worker has disconnected. If the node
// doesn't respond maxDisconnections time, the returned chan receives false.
func newDisconnectionWatchdog(w Worker, maxDisconnections int) chan bool {
	c := make(chan bool)
	var disconnections = 0

	go func() {
		for {
			time.Sleep(DefaultWatchdogSleep)

			if w.isOnline() {
				disconnections = 0
			} else {
				disconnections += 1

				if disconnections >= maxDisconnections {
					c <- true
				}
			}
		}
	}()

	return c
}