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
	"testing"
)

func TestWorkers_DistributeJobNoWorkers(t *testing.T) {
	s, _, _ := startPrimaryTestChannels()

	err := s.DistributeJob(getTestNodes(s), "", "")
	if err == nil {
		t.Fail()
		return
	}
}

// Getting this to work is tricky because an additional dependencies are needed for testing. Will implement later
/*
func TestWorkers_DistributeJob(t *testing.T) {
	workers := getTestNodes()
	_, sendChan := startPrimaryTestChannels()

	err := workers.DistributeJob("github.com/PLACEHOLDER/PLACEHOLDER", "PLACEHOLDER")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}

	var received int
	for len(workers) > received {
		select {
		case msg := <-sendChan:
			if msg.Operation != OperationJobTransfer {
				t.Fail()
				return
			}

			received += 1
		case <- time.After(time.Second):
			t.Fail()
			return
		}
	}
}
*/

func TestReadBinary(t *testing.T) {
	err := createFolderIfNotExist("./.beekeeper")
	if err != nil {
		t.Error(err)
		return
	}

	err = saveBinary("./.beekeeper/temp_windows", []byte("test"))
	if err != nil {
		t.Error(err)
		return
	}

	data, err := readBinary("./.beekeeper/temp_windows")
	if err != nil {
		t.Error(err)
		return
	}

	if bytes.Compare(data, []byte("test")) != 0 {
		t.Error(err)
		return
	}
}

func TestCleanupBuild(t *testing.T) {
	err := createFolderIfNotExist("./.beekeeper")
	if err != nil {
		t.Error(err)
		return
	}

	err = saveBinary("./.beekeeper/temp_windows", []byte("test"))
	if err != nil {
		t.Error(err)
		return
	}

	err = cleanupBuild()
	if err != nil {
		t.Error(err)
		return
	}

	if doesPathExists("./.beekeeper/temp_windows") {
		t.Error(err)
		return
	}
}
