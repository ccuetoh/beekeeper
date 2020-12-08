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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DistributeJob builds a job and sends a copy to the workers. Will fail if an empty workers list is given.
func (s *Server) DistributeJob(pkgName string, function string, nodes ...Node) error {
	if len(nodes) < 1 {
		return errors.New("no nodes provided")
	}

	n := Nodes(nodes)

	opSystems := n.getOperatingSystems()

	paths, err := buildJob(pkgName, function, opSystems)
	if err != nil {
		return err
	}

	if !s.Config.DisableConnectionWatchdog {
		terminateChan := make(chan bool, 1)
		go startConnectionWatchdog(s, terminateChan)
		defer func() {
			terminateChan <- true
		}()
	}

	binaries := make(map[string][]byte, len(opSystems))
	for _, opSys := range opSystems {
		data, err := readBinary(paths[opSys])
		if err != nil {
			return fmt.Errorf("unable to load binary for os %s: %s", opSys, err.Error())
		}

		binaries[opSys] = data
	}

	var binariesLock sync.RWMutex

	errChan := make(chan error, len(n))
	okChan := make(chan bool, len(n))

	for _, node := range n {
		go func(node Node) {
			binariesLock.RLock()
			data := binaries[node.Info.OS]
			binariesLock.RUnlock()

			msg := Message{
				Operation: OperationJobTransfer,
				Data:      data,
			}

			err = s.send(node, msg)
			if err != nil {
				errChan <- fmt.Errorf("unable to sendWithConn job to node %s: %s", node.Name, err.Error())
			}

			err = s.awaitTransfer(node)
			if err != nil {
				if err == ErrNodeDisconnected {
					errChan <- fmt.Errorf("unable to sendWithConn job to node %s: node disconnected", node.Name)
				}

				errChan <- fmt.Errorf("unable to sendWithConn job to node %s: %s", node.Name, err)
			}

			okChan <- true
		}(node)
	}

	okays := 0
	for okays < len(n) {
		select {
		case <-okChan:
			okays += 1
		case err := <-errChan:
			return err
		}
	}

	if !s.Config.DisableCleanup {
		err = cleanupBuild()
		if err != nil {
			log.Println("Unable to perform cleanup:", err.Error())
		}
	}

	return nil
}

// cleanupBuild removes build files and binaries.
func cleanupBuild() error {
	folderPath := filepath.FromSlash("./.beekeeper")
	if !doesPathExists(folderPath) {
		return nil // Nothing to do here
	}

	// Remove temp.go
	tempGoFile := filepath.FromSlash(folderPath + "/temp.go")
	if doesPathExists(tempGoFile) {
		err := os.Remove(tempGoFile)
		if err != nil {
			return err
		}
	}

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	// Remove all matching temp_GOOS binaries
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasPrefix(file.Name(), "temp_") {
			err := os.Remove(filepath.FromSlash(folderPath + "/" + file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// readBinary reads the binary file on the given path and returns a byte slice with its content. It will fail if the
// file does not exists or is busy.
func readBinary(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	stats, statsErr := f.Stat()
	if statsErr != nil {
		return nil, statsErr
	}

	var size = stats.Size()
	buf := make([]byte, size)

	bufR := bufio.NewReader(f)
	_, err = bufR.Read(buf)

	return buf, err
}

// saveBinary creates or replaces a file in the given path and writes a byte slice to its contents.
func saveBinary(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	return nil
}

// createFolderIfNotExist checks if a folder exists in the given path. If none is found one is created.
func createFolderIfNotExist(path string) error {
	if !doesPathExists(path) {
		err := os.Mkdir(path, 0600)
		if err != nil {
			return err
		}
	}

	return nil
}

// doesPathExists checks whether a file or folder exists in the given path.
func doesPathExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return true
}
