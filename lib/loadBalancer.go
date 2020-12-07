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
	"math"
	"math/rand"
	"sync"
	"time"
)

// LoadBalancer contains the data needed to try to select the best node for a task.
// Should be created using NewLoadBalancer.
type LoadBalancer struct {
	server  *Server
	best    int64
	records nodeRecords
	lock    sync.Mutex
}

type nodeRecords []*nodeRecord

type nodeRecord struct {
	node   Node
	record record
}

type record struct {
	load int
	time int64
}

// NewLoadBalancer creates and sets up a LoadBalancer from the given Nodes.
func NewLoadBalancer(s *Server, ns Nodes) *LoadBalancer {
	var records []*nodeRecord

	for _, w := range ns {
		records = append(records, &nodeRecord{node: w, record: record{time: time.Second.Milliseconds()}})
	}

	return &LoadBalancer{records: records, best: time.Hour.Milliseconds(), server: s}
}

// Execute will run a task, selecting the node based on it's workload. If multiple nodes are equally as busy, the
// LoadBalancer will pick the best performing one, or pick based on a Softmax algorithm for exploration.
func (lb *LoadBalancer) Execute(t Task, timeout ...time.Duration) (res Result, err error) {
	lb.lock.Lock()

	use := lb.pick()

	use.record.load += 1
	defer func() {
		lb.lock.Lock()
		use.record.load -= 1
		lb.lock.Unlock()
	}()

	lb.lock.Unlock()

	start := time.Now()
	res, err = lb.server.Execute(use.node, t, timeout...)
	if err != nil {
		return Result{}, err
	}

	use.record.time = time.Since(start).Milliseconds()
	if use.record.time < lb.best {
		lb.best = use.record.time
	}

	return res, nil
}

// getLowestLoad runs through a slice of nodeRecords and returns the lowes loaded ones. On a tie all the tied nodes
// are returned.
func (rs nodeRecords) getLowestLoad() nodeRecords {
	var records nodeRecords
	var lowest = 999999

	for _, wr := range rs {
		if wr.record.load <= lowest {
			lowest = wr.record.load
			records = append(records, wr)
		}
	}

	return records
}

// pick selects the best node based on load, performance or a Softmax algorithm depending on the case.
func (lb *LoadBalancer) pick() *nodeRecord {
	rand.Seed(time.Now().UTC().UnixNano())

	softmax := lb.records.getLowestLoad().softmax(lb.best)
	for {
		for i, prob := range softmax {
			if prob > rand.Float64() {
				return lb.records[i]
			}
		}
	}
}

// softmax implements the Softmax algorithm to give the distributions of a nodeRecords object based on performance as
// measured by time of execution.
func (rs nodeRecords) softmax(best int64) []float64 {
	var max = float64(rs[0].record.time / best)
	for _, r := range rs {
		max = math.Max(max, float64(r.record.time/best))
	}

	a := make([]float64, len(rs))

	var sum float64 = 0
	for i, r := range rs {
		a[i] -= math.Exp(float64(r.record.time/best) - max)
		sum += a[i]
	}

	for i, n := range a {
		a[i] = n / sum
	}

	return a
}
