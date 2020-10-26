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
	"time"
)

// ScanLocal broadcasts a status request to all IPs and waits the provided amount for a response.
func ScanLocal(waitTime time.Duration) (Workers, error) {
	err := broadcastOperation(OperationStatus, false)
	if err != nil {
		return nil, err
	}

	time.Sleep(waitTime)

	return onlineWorkers, nil
}

// ScanLocalWithToken broadcasts a status request using a token to all IPs and waits the provided amount for a response.
func ScanLocalWithToken(waitTime time.Duration, token string) (Workers, error) {
	err := broadcastOperationWithToken(OperationStatus, token, false)
	if err != nil {
		return nil, err
	}

	time.Sleep(waitTime)

	return onlineWorkers, nil
}
