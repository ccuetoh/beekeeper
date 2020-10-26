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
)

// DistributeJob builds a job and sends a copy to the workers. Will fail if an empty workers list is given.
func (w Workers) DistributeJob(pkgName string, function string) error {
	if len(w) < 1 {
		return errors.New("no workers provided")
	}

	opSys := w.getOperatingSystems()

	paths, err := buildJob(pkgName, function, opSys)
	if err != nil {
		return err
	}

	for _, worker := range w {
		data, err := readBinary(paths[worker.Info.OS])
		if err != nil {
			return errors.New(fmt.Sprintf("unable to load binary for os %s: %s", worker.Info.OS, err.Error()))
		}

		msg := Message{
			Operation: OperationJobTransfer,
			Data:      data,
		}

		err = worker.send(msg)
		if err != nil {
			return errors.New(fmt.Sprintf("unable to send job to worker %s: %s", worker.Name, err.Error()))
		}

		res := awaitTransferAndCheck(worker, 2)
		if res != "" {
			return errors.New(fmt.Sprintf("unable to send job to worker %s, %s", worker.Name, res))
		}
	}

	if !mySettings.Config.DisableCleanup {
		err = cleanupBuild()
		if err != nil {
			log.Println("Unable to perform cleanup:", err.Error())
		}
	}

	return nil
}

// cleanupBuild removes build files and binaries.
func cleanupBuild() error {
	sep := string(filepath.Separator)

	folderPath := "." + sep + ".beekeeper"
	if !doesPathExists(folderPath) {
		return nil // Nothing to do here
	}

	// Remove temp.go
	tempGoFile := folderPath + sep + "temp.go"
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
			err := os.Remove(folderPath + sep + file.Name())
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
		err := os.Mkdir(path, 0700)
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
