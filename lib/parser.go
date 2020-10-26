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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// buildTemplate is a small Go program template that wraps a job into WrapJob.
const buildTemplate = `package main

import (
	"github.com/CamiloHernandez/beekeeper/lib"
	p "%s"
)

func main() {
	beekeeper.WrapJob(p.%s)
}

`

// buildJob creates a wrapped implementation of the given function and builds for every GOOS in the
// distributions parameter. It returns a map containing the GOOSes and their executable's paths.
func buildJob(pkgName string, function string, distributions []string) (map[string]string, error) {
	content := []byte(generateBuildFile(pkgName, function))

	sep := string(filepath.Separator)

	outPath := "." + sep + ".beekeeper"
	filePath := outPath + sep + "temp.go"

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		err = os.Mkdir(outPath, 0700)
		if err != nil {
			return nil, err
		}
	}

	err := ioutil.WriteFile(filePath, content, 0700)
	if err != nil {
		return nil, err
	}

	binPaths := make(map[string]string)
	for _, goos := range distributions {
		log.Println("Building binaries for", goos)

		err = os.Setenv("GOOS", goos)
		if err != nil {
			return nil, err
		}

		outFile := outPath + sep + "temp_" + goos

		cmd := exec.Command("go", "build", "-o", outFile, "-ldflags", "-s -w", filePath)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, errors.New("go build error: " + string(out))
		}

		binPaths[goos] = outFile
	}

	return binPaths, nil
}

// generateBuildFile formats the passed pkgName and funcName.
func generateBuildFile(pkgName, funcName string) string {
	return fmt.Sprintf(buildTemplate, pkgName, funcName)
}
