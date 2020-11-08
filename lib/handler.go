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
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
)

// defaultHandler will process a TCPConnection and return a Message object with its data if possible. Connections
// coming form the host machine are discarded.
func defaultHandler(c chan Message, conn net.Conn) {
	isPipe := conn.RemoteAddr().String() == "pipe"

	if conn.RemoteAddr() == conn.LocalAddr() && !isPipe {
		return
	}

	reader := bufio.NewReader(conn)
	header, _, err := reader.ReadLine()
	if err != nil {
		log.Println("Error reading connection header:", err.Error())
		_ = conn.Close()
		return
	}

	dataLen, err := strconv.Atoi(string(header))
	if err != nil {
		log.Println("Error parsing connection header:", err.Error())
		_ = conn.Close()
		return
	}

	dataBuf := make([]byte, dataLen)

	readLen, err := io.ReadFull(reader, dataBuf)
	if err != nil {
		log.Println("Error reading connection:", err.Error())
		_ = conn.Close()
		return
	}

	if readLen != dataLen {
		log.Printf("Error: Expected to read %d bytes, but read %d\n", readLen, dataLen)
		_ = conn.Close()
		return
	}

	msg, err := decodeMessage(dataBuf)
	if err != nil {
		log.Println("Error reading data:", err.Error())
		_ = conn.Close()
		return
	}

	if !isPipe {
		tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
		msg.Addr = tcpAddr
	}

	c <- msg
}
