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

// Package cmd provides the command-line interfaces to build the Bee CLI Tool
package cmd

import (
	"fmt"
	"github.com/CamiloHernandez/beekeeper/lib"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [-p port] [-t token] [-c config]",
	Short: "Start a new Beekeeper server on the machine",
	Long: `A new Beekeeper server is created as a node. Unless
configured otherwise the default port 2020 and no token is used. No more than one
server might be running on the same machine.

For a detailed usage guide visit https://www.beekeeper.dev`,
	Run: func(cmd *cobra.Command, args []string) {
		instanceCfg := cfg
		if portOverride != 0 {
			instanceCfg.InboundPort = portOverride
		}

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		sv := beekeeper.NewServer(instanceCfg)

		go func() {
			<-c
			log.Println("Shutting down server")
			sv.Stop()
			os.Exit(0)
		}()

		err := sv.Start()
		if err != nil {
			fmt.Println("Unable to start server:", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
