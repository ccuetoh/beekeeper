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
	"encoding/gob"
)

// Task is used to run a job. In order to create a Task use NewTask; not this structure directly.
type Task struct {
	UUID      string
	Arguments map[string]interface{}
	Returns   map[string]interface{}
	Error     string
}

// NewTask creates a Task, initializes and then returns it.
func NewTask() Task {
	return Task{
		UUID:      "",
		Arguments: make(map[string]interface{}),
		Returns:   make(map[string]interface{}),
		Error:     "",
	}
}

// encode returns a gob encoded Task.
func (t Task) encode() ([]byte, error) {
	var buf bytes.Buffer

	// There is some debate on whether creating an encoder everytime is a good idea
	// but Reddit says it's ok:
	// https://www.reddit.com/r/golang/comments/7ospor/gob_encoding_how_do_you_use_it_in_production/
	gobEncoder := gob.NewEncoder(&buf)

	err := gobEncoder.Encode(t)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decodeTask decodes a gob encoded task.
func decodeTask(data []byte) (Task, error) {
	buf := bytes.NewBuffer(data)

	gobDecoder := gob.NewDecoder(buf)

	task := Task{}
	err := gobDecoder.Decode(&task)
	if err != nil {
		return Task{}, err
	}

	return task, nil
}
