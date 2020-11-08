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
	"net"
	"sync"
	"time"
)

// awaitedTasks holds a list of the task waiting for results and the corresponding notification channels.
var awaitedTasks = make(map[string]chan Result)

// awaitedTasksLock blocks the awaitedTasks for asynchronous use.
var awaitedTasksLock sync.Mutex

// awaitedTransfers holds a list of the IPs whose transfers are waiting for confirmations and the corresponding
// notification channel.
var awaitedTransfers = make(map[*net.TCPAddr]chan string)

// awaitedTransfersLock blocks the awaitedTransfers for asynchronous use
var awaitedTransfersLock sync.Mutex

var ErrTimeout = errors.New("time exceeded")
var ErrNodeDisconnected = errors.New("node disconnected")

// awaitTaskWithTimeout blocks the execution until a node sends a Result with a matching taskID or the assigned time
// window is expired.
func awaitTaskWithTimeout(taskID string, timeout time.Duration) (Result, error) {
	resChan := make(chan Result)

	awaitedTasksLock.Lock()
	awaitedTasks[taskID] = resChan
	awaitedTasksLock.Unlock()

	select {
	case res := <-resChan:
		return res, nil
	case <-time.After(timeout):
		return Result{}, ErrTimeout
	}
}

// awaitTask blocks the execution until a node sends a Result with a matching taskID.
func awaitTask(taskId string) Result {
	resChan := make(chan Result)

	awaitedTasksLock.Lock()
	awaitedTasks[taskId] = resChan
	awaitedTasksLock.Unlock()

	return <-resChan
}

// processAwaitedTask compares a Result object with the awaited tasks. If a match is found the Result is passed forward
// to the assigned chan and the task is deleted from the awaited tasks list.
func processAwaitedTask(res Result) bool {
	awaitedTasksLock.Lock()
	defer awaitedTasksLock.Unlock()

	for taskID, c := range awaitedTasks {
		if taskID == res.UUID {
			c <- res
			delete(awaitedTasks, taskID)

			return true
		}
	}

	return false
}

// awaitTransfer blocks the execution until the worker sends a transfer acknowledgement or reports a transfer error.
// If an error message is received i'll be returned. An empty string means no error was raised.
func awaitTransfer(w Worker) error {
	errChan := make(chan string)

	awaitedTransfersLock.Lock()
	awaitedTransfers[w.Addr] = errChan
	awaitedTransfersLock.Unlock()

	res := <-errChan
	if res == "" {
		return nil
	}

	return errors.New(res)
}

// awaitTransferAndCheck blocks the execution until the worker sends a transfer acknowledgement or reports a transfer
// error. If an error message is received i'll be returned. It will create a thread to check if the worker is still
// responding to Status operations, and if no response is received more than maxDisconnect times, the transfer will be
// considered failed.
func awaitTransferAndCheck(w Worker, maxDisconnect int) error {
	successChan := make(chan error)

	// Result routine
	go func() {
		successChan <- awaitTransfer(w)
	}()

	disconnectChan := newDisconnectionWatchdog(w, maxDisconnect)

	select {
	case <-disconnectChan:
		return ErrNodeDisconnected
	case errMsg := <-successChan:
		return errMsg
	}
}

// processAwaitedTask compares a Message object with the awaited transfers. If a match is found the transfer Result
// is passed forward to the assigned chan and the transfer is deleted from the awaited transfer list.
func processAwaitedTransfer(msg Message) bool {
	awaitedTransfersLock.Lock()
	defer awaitedTransfersLock.Unlock()

	for ip, c := range awaitedTransfers {
		if msg.Addr.IP.Equal(ip.IP) {
			data := string(msg.Data)
			if data == "" && msg.Operation == OperationTransferFailed {
				data = "no further explanation received"
			}

			c <- data
			delete(awaitedTransfers, ip)

			return true
		}
	}

	return false
}
