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

package cmd

import (
	"fmt"
	"github.com/CamiloHernandez/beekeeper/lib"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var cfgFilePath string

var tokenOverride string
var cleanupOverride bool
var debugOverride bool
var portOverride int

var cfg beekeeper.Config

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bee [command]",
	Short: "Batteries-included cluster computing in Go",
	Long: `Beekeeper is a batteries-included distributed and cluster computing library for the Go programming language.
This is the auxiliary CLI tool, that allows the easy creation and monitoring of nodes. 

For detailed usage instructions visit https://www.beekeeper.dev 

The Beekeeper CLI Tool, and the Beekeeper library are released as open-source under the MIT licence. (c) Camilo Hernández 2020`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init sets the flags for rootCmd.
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFilePath, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVarP(&tokenOverride, "token", "t", "", "sets a token")
	rootCmd.PersistentFlags().BoolVarP(&cleanupOverride, "cleanup", "c", true, "enables post-build cleanup")
	rootCmd.PersistentFlags().BoolVar(&debugOverride, "debug", false, "enables debug mode")
	rootCmd.PersistentFlags().IntVarP(&portOverride, "port", "p", 0, "sets a custom port")
}

// initConfig reads in the config file and manages the persistent flags.
func initConfig() {
	cfg = findConfig(cfgFilePath)

	if cleanupOverride {
		cfg.DisableCleanup = true
	}

	if debugOverride {
		cfg.Debug = true
	}

	if tokenOverride != "" {
		cfg.Token = tokenOverride
	}

	return
}

// findConfig will use a custom config file if set, and if none is provided will try to find a matching file. If none of
// the adobe, a default config is returned
func findConfig(path string) beekeeper.Config {
	if path != "" {
		config, err := beekeeper.NewConfigFromFile(cfgFilePath)
		if err != nil {
			log.Println("Unable to use config file, using default values:", err.Error())
			config = beekeeper.NewDefaultConfig()
		}

		return config
	}

	ex, err := os.Executable()
	if err != nil {
		return beekeeper.NewDefaultConfig()
	}

	folderPath := filepath.Dir(ex)
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return beekeeper.NewDefaultConfig()
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := filepath.Base(file.Name())

		if strings.HasPrefix(fileName, "beekeeper.") {
			config, err := beekeeper.NewConfigFromFile(folderPath + string(filepath.Separator) + file.Name())
			if err != nil {
				config = beekeeper.NewDefaultConfig()
			}

			return config
		}
	}

	return beekeeper.NewDefaultConfig()
}
