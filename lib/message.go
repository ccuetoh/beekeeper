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
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"time"
)

// Operation is used to specify a Message's intent to the remote node
type Operation int

const (
	// OperationNone nil value for operations
	OperationNone = iota

	// OperationStatus ask a node for a status report
	OperationStatus

	// OperationJobTransfer transfer a job via the Data field
	OperationJobTransfer

	// OperationTransferFailed transfer failed, Data contains the details
	OperationTransferFailed

	// OperationTransferAcknowledge transfer was successful
	OperationTransferAcknowledge

	// OperationJobExecute run the local job
	OperationJobExecute

	// OperationJobResult job ran and the details come in the Data
	OperationJobResult
)

// String returns a string representation of the Operation.
func (o Operation) String() string {
	return []string{"None", "Status", "JobTransfer", "JobTransferFailed",
		"JobTransferAcknowledge", "JobExecute", "JobResult"}[o]
}

// Message is used for node communication. It holds the transferable data as well as some metadata about the node.
type Message struct {
	SentAt        time.Time
	From          string
	Operation     Operation
	Data          []byte
	Token         string
	Addr          *net.TCPAddr
	RespondOnPort int
	Status        Status
	NodeInfo      NodeInfo
}

// NodeInfo holds additional info abut a node.
type NodeInfo struct {
	CPUTemp float32
	Usage   float32
	OS      string
}

func newMessage() Message {
	return Message{Addr: &net.TCPAddr{}}
}

// encode returns a gob encoded and gzip compressed message.
func (m Message) encode() ([]byte, error) {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	gobEncoder := gob.NewEncoder(gzipWriter)

	err := gobEncoder.Encode(m)
	if err != nil {
		return nil, err
	}

	err = gzipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// node uses the Message's metadata to construct a node object.
func (m Message) node() Node {
	return Node{
		Addr:   m.Addr,
		Name:   m.From,
		Status: m.Status,
		Info:   m.NodeInfo,
	}
}

// summary returns a string with relevant information about the Message.
func (m Message) summary() string {
	var addr string
	if m.Addr != nil {
		addr = m.Addr.IP.String()
	} else {
		addr = "?"
	}

	return fmt.Sprintf("[Sender: %s (%s), Opearation: %s, Data size: %d]",
		addr, m.From, m.Operation.String(), len(m.Data))
}

// respond creates a new Conn and sends a Message through it.
func (m Message) respond(s *Server, response Message) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error while responding to node %s\n", m.From)
		}
	}()

	var addr string
	if m.RespondOnPort != 0 {
		addr = fmt.Sprintf("%s:%d", m.Addr.IP.String(), m.RespondOnPort)
	} else {
		addr = m.Addr.IP.String()
	}

	conn, err := s.connect(addr)
	if err != nil {
		return err
	}

	defer conn.Close()

	err = conn.send(response)
	if err != nil {
		return err
	}

	return nil
}

// isTokenMatching compares the a Message's token to the one present in the local node info and returns whether it's
// matching or not.
func (m Message) isTokenMatching(token2 string) bool {
	if m.Token == token2 {
		return true
	}

	return false
}

// decodeMessage expects a byte slice with a gob encoded and gzip compressed message data and turns it into a
// Message object.
func decodeMessage(data []byte) (Message, error) {
	buf := bytes.NewBuffer(data)

	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return Message{}, err
	}

	gobDecoder := gob.NewDecoder(gzipReader)

	msg := Message{}
	err = gobDecoder.Decode(&msg)
	if err != nil {
		return Message{}, err
	}

	return msg, nil
}

func (m Message) setData(data interface{}) (Message, error) {
	var buf bytes.Buffer

	gobEncoder := gob.NewEncoder(&buf)
	err := gobEncoder.Encode(data)
	if err != nil {
		return Message{}, err
	}

	m.Data = buf.Bytes()

	return m, nil
}
