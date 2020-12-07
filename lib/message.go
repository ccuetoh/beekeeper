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
	// SentAt timestamp for the Message.
	SentAt time.Time

	// Name the sender's name.
	Name string

	// Operation operation the remote node wishes to execute. It may be nilled with OperationNone.
	Operation Operation

	// Data the body of the message. Contains the payload needed for the execution if the Operation.
	Data []byte

	// Token is used as a passphrase to operate in a multi-node environment.
	Token string

	// Addr is the address of the sender
	Addr *net.TCPAddr

	// RespondOnPort is the port that the sender wishes to be used for the response.
	RespondOnPort int

	// Status represents the current action the node is doing.
	Status Status

	// NodeInfo contains metadata about the sender, like OS and current usage.
	NodeInfo NodeInfo
}

// NodeInfo holds additional info abut a node.
type NodeInfo struct {
	// CPUTemp is the temperature as measured in the CPU dice when possible. Certain OS can return 0.
	CPUTemp float32

	// Usage is the percentage of usage of the host system in a range from 1 (max) to 0 (min).
	Usage float32

	// OS is the GOOS of the host system.
	OS string
}

// newMessage creates an empty message with a non-nil address
func newMessage() Message {
	return Message{Addr: &net.TCPAddr{}}
}

// encode returns a gob encoded and gzip compressed message.
func (m Message) encode() ([]byte, error) {
	var buf bytes.Buffer

	// There is some debate on whether creating an encoder everytime is a good idea
	// but Reddit says it's ok:
	// https://www.reddit.com/r/golang/comments/7ospor/gob_encoding_how_do_you_use_it_in_production/
	gzipWriter := gzip.NewWriter(&buf)
	gobEncoder := gob.NewEncoder(gzipWriter)

	err := gobEncoder.Encode(m)
	if err != nil {
		return nil, err
	}

	_ = gzipWriter.Close()

	return buf.Bytes(), nil
}

// node uses the Message's metadata to construct a node object.
func (m Message) node() Node {
	return Node{
		Addr:   m.Addr,
		Name:   m.Name,
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
		addr, m.Name, m.Operation.String(), len(m.Data))
}

// respond creates a new Conn and sends a Message through it.
func (m Message) respond(s *Server, response Message) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error while responding to node %s\n", m.Name)
		}
	}()

	var addr string
	if m.RespondOnPort != 0 {
		addr = fmt.Sprintf("%s:%d", m.Addr.IP.String(), m.RespondOnPort)
	} else {
		addr = m.Addr.IP.String()
	}

	conn, err := s.dial(addr)
	if err != nil {
		return err
	}

	err = s.sendWithConn(conn, response)
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
