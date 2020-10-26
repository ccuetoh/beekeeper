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

// Package beekeeper is a batteries-included cluster computing library
package beekeeper

import (
	"bytes"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io"
	"net"
	"sort"
	"time"
)

// Worker represents a worker node.
type Worker struct {
	Addr   *net.TCPAddr
	Name   string
	OS     string
	Status Status
	Info   NodeInfo
}

// Equals compares two workers. The comparison is made using the IP addresses of the nodes.
func (w Worker) Equals(w2 Worker) bool {
	return w.Addr.IP.Equal(w2.Addr.IP)
}

// isOnline searches the worker in the onlineWorkers slice
func (w Worker) isOnline() bool {
	onlineWorkersLock.Lock()
	defer onlineWorkersLock.Unlock()

	for _, worker := range onlineWorkers {
		if w.Equals(worker) {
			return true
		}
	}

	return false
}

// send creates a new nodeConn and sends the provided Message.
func (w Worker) send(m Message) error {
	conn, err := newNodeConn(w.Addr.IP.String())
	if err != nil {
		return err
	}

	err = conn.send(m)
	if err != nil {
		return err
	}

	err = conn.Close()
	if err != nil {
		return err
	}

	return nil
}

// Workers is a Worker slice
type Workers []Worker

// getOperatingSystems iterates the workers and returns a set of the GOOSs found.
func (w Workers) getOperatingSystems() (opSys []string) {
	for _, worker := range w {
		duplicate := false

		for _, ops := range opSys {
			if ops == worker.Info.OS {
				duplicate = true
			}
		}

		if !duplicate {
			opSys = append(opSys, worker.Info.OS)
		}
	}

	return opSys
}

// PrettyPrint prints a formatted table of workers.
func (w Workers) PrettyPrint(writer io.Writer) {
	table := tablewriter.NewWriter(writer)

	table.SetHeader([]string{"Name", "Address", "Status"})
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	for _, worker := range w {
		table.Append([]string{worker.Name, worker.Addr.IP.String(), worker.Status.String()})
	}

	table.Render()
}

// update adds new workers if not present and replaces old ones if matching.
func (w Workers) update(newWorker Worker) Workers {
	onlineWorkersLock.Lock()
	defer onlineWorkersLock.Unlock()

	for i, worker := range w {
		if worker.Addr.IP.String() == newWorker.Addr.IP.String() {
			w[i] = newWorker
			return w
		}
	}

	return append(w, newWorker)
}

// Execute runs a task on the provided Workers and blocks until a Result is sent back. Optionally a timeout
// argument can be passed.
func (w Workers) Execute(t Task, timeout ...time.Duration) ([]Result, error) {
	resultsChan := make(chan Result)
	errChan := make(chan error)

	for _, worker := range w {
		go func(worker Worker, rc chan Result, ec chan error) {
			res, err := worker.Execute(t, timeout...)
			if err != nil {
				ec <- fmt.Errorf("worker %s error: %s", worker.Name, err.Error())
			} else {
				rc <- res
			}
		}(worker, resultsChan, errChan)
	}

	var results []Result

	for len(results) != len(w) {
		select {
		case err := <-errChan:
			return nil, err

		case res := <-resultsChan:
			results = append(results, res)
		}
	}

	return results, nil
}

// sort orders a slice of workers based on their IP address.
func (w Workers) sort() Workers {
	sort.Slice(w, func(i, j int) bool {
		return bytes.Compare(w[i].Addr.IP, w[j].Addr.IP) < 0
	})

	return w
}
