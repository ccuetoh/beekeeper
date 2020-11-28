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
	"log"
	"net"
	"os"
	"sort"
	"time"
)

// Node represents a node node.
type Node struct {
	Conn   *Conn
	Addr   *net.TCPAddr
	Name   string
	Status Status
	Info   NodeInfo
	server *Server
}

// Nodes is a Node slice
type Nodes []Node

// Equals compares two workers. The comparison is made using the IP addresses of the nodes.
func (n Node) Equals(w2 Node) bool {
	return n.Addr.IP.Equal(w2.Addr.IP)
}

// send sends the provided Message.
func (n Node) send(m Message) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error while sending to node %s: %s\n", m.From, r)
		}
	}()

	if n.Conn == nil {
		var err error
		n.Conn, err = n.server.dial(n.Addr.IP.String())
		if err != nil {
			return err
		}
	}

	err := n.Conn.send(m)
	if err != nil {
		return err
	}

	return nil
}

// getOperatingSystems iterates the workers and returns a set of the GOOSs found.
func (n Nodes) getOperatingSystems() (opSys []string) {
	for _, node := range n {
		duplicate := false

		for _, ops := range opSys {
			if ops == node.Info.OS {
				duplicate = true
			}
		}

		if !duplicate {
			opSys = append(opSys, node.Info.OS)
		}
	}

	return opSys
}

// PrettyPrint prints a formatted table of workers.
func (n Nodes) PrettyPrint(writer ...io.Writer) {
	var out io.Writer
	if len(writer) > 0 {
		out = writer[0]
	} else {
		out = os.Stdout
	}

	table := tablewriter.NewWriter(out)

	table.SetHeader([]string{"Name", "Address", "Status"})
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	for _, node := range n {
		table.Append([]string{node.Name, node.Addr.IP.String(), node.Status.String()})
	}

	table.Render()
}

// updateNode adds new workers if not present and replaces old ones if matching.
func (s *Server) updateNode(node2 Node) {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node2.server = s

	for i, node := range s.nodes {
		if node.Addr.IP.Equal(node2.Addr.IP) {
			s.nodes[i] = node2
			return
		}
	}

	s.nodes = append(s.nodes, node2)
}

// Execute runs a task on the provided Nodes and blocks until a Result is sent back. Optionally a timeout
// argument can be passed.
func (n Nodes) Execute(t Task, timeout ...time.Duration) ([]Result, error) {
	resultsChan := make(chan Result)
	errChan := make(chan error)

	for _, node := range n {
		go func(node Node, rc chan Result, ec chan error) {
			res, err := node.Execute(t, timeout...)
			if err != nil {
				ec <- fmt.Errorf("node %s error: %s", node.Name, err.Error())
			} else {
				rc <- res
			}
		}(node, resultsChan, errChan)
	}

	var results []Result

	for len(results) != len(n) {
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
func (n Nodes) sort() Nodes {
	sort.Slice(n, func(i, j int) bool {
		return bytes.Compare(n[i].Addr.IP, n[j].Addr.IP) < 0
	})

	return n
}

// find orders a slice of workers based on their IP address.
func (n Nodes) find(addr net.IP) Node{
	for _, node := range n {
		if node.Addr.IP.Equal(addr) {
			return node
		}
	}

	return Node{}
}
