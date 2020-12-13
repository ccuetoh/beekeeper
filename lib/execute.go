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
	"errors"
	"github.com/sony/sonyflake"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// flake holds a SonyFlake object for UUID creation. The start time is time.Now().
var flake = newFlake()

// Execute runs a task on the given node and blocks until the task results are retrieved.
// It will fail if no job is present on the node's systems. An optional timeout parameter can be provided.
func (s *Server) Execute(n Node, t Task, timeout ...time.Duration) (res Result, err error) {
	if !s.Config.DisableConnectionWatchdog {
		terminateChan := make(chan bool, 1)
		go startConnectionWatchdog(s, terminateChan)
		defer func() {
			terminateChan <- true
		}()
	}

	t.UUID, err = newJobUUID()
	if err != nil {
		return Result{}, err
	}

	data, err := t.encode()
	if err != nil {
		return Result{}, err
	}

	err = s.send(n, Message{
		Operation: OperationJobExecute,
		Data:      data,
	})
	if err != nil {
		return Result{}, err
	}

	res, err = s.awaitTask(t.UUID, timeout...)
	if err != nil {
		return Result{}, err
	}

	if res.Error != "" {
		return Result{}, errors.New(res.Error)
	}

	return res, nil
}

// runLocalJob will execute the current job on the beekeeper folder. Fails if no job is present.
func runLocalJob(t Task) (Result, error) {
	data, err := t.encode()
	if err != nil {
		return Result{}, err
	}

	path := filepath.FromSlash("./.beekeeper/job.bin")
	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Result{}, errors.New("unable to get stdin pipe: " + err.Error())
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, errors.New("unable to get stdout pipe: " + err.Error())
	}

	err = cmd.Start()
	if err != nil {
		return Result{}, errors.New("unable to start process: " + err.Error())
	}

	_, err = stdin.Write(append(data, byte('\n')))
	if err != nil {
		return Result{}, errors.New("unable to write task to process: " + err.Error())
	}

	reader := bufio.NewReader(stdout)

	header, _, err := reader.ReadLine()
	if err != nil {
		return Result{}, errors.New("error reading data header: " + err.Error())
	}

	dataLen, err := strconv.Atoi(string(header))
	if err != nil {
		return Result{}, errors.New("error parsing data header: " + err.Error())
	}

	dataBuf := make([]byte, dataLen)

	_, err = io.ReadFull(reader, dataBuf)
	if err != nil {
		return Result{}, errors.New("unable to read data from process: " + err.Error())
	}

	res, err := decodeResult(dataBuf)
	if err != nil {
		return Result{}, err
	}

	res.UUID = t.UUID

	return res, nil
}

// newFlake creates a new SonyFlake generator. If the instantiation of the generator fails, a randomly generated one
// is provided. If both options fail it exists.
func newFlake() *sonyflake.Sonyflake {
	s := sonyflake.Settings{StartTime: time.Now()}
	f := sonyflake.NewSonyflake(s)
	if f != nil {
		// Normal creation was ok
		return f
	}

	// Flake was unable to start, use a random generator
	log.Println("Unable to start the UUID generator, a random generator will be used.")

	s.MachineID = func() (uint16, error) {
		return uint16(rand.Uint32()), nil
	}

	s.CheckMachineID = func(_ uint16) bool {
		return true
	}

	f = sonyflake.NewSonyflake(s)
	if f != nil {
		return f
	}

	// All options failed
	log.Fatalln("Unable to start UUID or random generator.")
	return nil
}

// newJobUUID creates a new UUID for job identification. It's not guaranteed to be unique for multiple sessions.
func newJobUUID() (string, error) {
	num, err := flake.NextID()
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(num, 24), nil
}
