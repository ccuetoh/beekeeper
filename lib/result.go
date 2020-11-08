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
	"fmt"
	"io"
	"os"
)

// Result holds the details from a job execution.
type Result struct {
	UUID  string
	Task  Task
	Error string
}

// newErrorResult creates an empty Result with Error set to err.
func newErrorResult(err error) Result {
	return Result{
		Error: err.Error(),
	}
}

// encode returns a gob encoded byte slice representing the Result.
func (r Result) encode() ([]byte, error) {
	var buf bytes.Buffer

	gobEncoder := gob.NewEncoder(&buf)

	err := gobEncoder.Encode(r)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decodeResult returns a result from a gob encoded byte slice.
func decodeResult(data []byte) (Result, error) {
	buf := bytes.NewBuffer(data)

	gobDecoder := gob.NewDecoder(buf)

	res := Result{}
	err := gobDecoder.Decode(&res)
	if err != nil {
		return Result{}, err
	}

	return res, nil
}

// printEncode encodes the Result and header and prints it to stdio.
func (r Result) printEncode(output ...io.Writer) {
	var out io.Writer
	if len(output) > 0 {
		out = output[0]
	} else {
		out = io.Writer(os.Stdout)
	}

	data, err := r.encode()
	if err != nil {
		_, _ = fmt.Fprintln(out, "FATAL: "+err.Error())
	}

	header := []byte(fmt.Sprintf("%d\n", len(data)))
	data = append(header, data...)

	_, _ = fmt.Fprint(out, string(data))
}
