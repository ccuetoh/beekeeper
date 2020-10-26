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
	"github.com/spf13/viper"
	"log"
	"time"
)

const (
	DefaultPort          = 2020 // DefaultPort is the default port for Beekeeper servers
	DefaultScanTime      = time.Second * 2 // DefaultScanTime is the scan time to be used by scan functions
	DefaultWatchdogSleep = time.Second * 15 // DefaultWatchdogSleep is the time between node pings for the watchdog
)

// Config holds the configurations for a worker or a primary node.
type Config struct {
	Name                      string `mapstructure:"name,omitempty"`
	Debug                     bool   `mapstructure:"debug,omitempty"`
	Token                     string `mapstructure:"token,omitempty"`
	InboundPort               int    `mapstructure:"inbound_port,omitempty"`
	OutboundPort              int    `mapstructure:"outbound_port,omitempty"`
	DisableCleanup            bool   `mapstructure:"disable_cleanup,omitempty"`
	DisableConnectionWatchdog bool   `mapstructure:"disable_connection_watchdog,omitempty"`
}

// NewDefaultConfig returns a new Config with sensible defaults. It's recommended that NewDefaultConfig be used.
// for the creation of Config structs.
func NewDefaultConfig() (c Config) {
	name, err := getHostname()
	if err != nil {
		log.Println("Error while fetching computer name:", err.Error())
	} else {
		c.Name = name
	}

	c.InboundPort = DefaultPort
	c.OutboundPort = DefaultPort
	c.DisableCleanup = true

	return c
}

// NewConfigFromFile parses a file on the provided path as a Config object. If a field is not set, the default value
// is assigned.
func NewConfigFromFile(path string) (c Config, err error) {
	if path != "" {
		viper.SetConfigFile(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	config := NewDefaultConfig()

	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, err
	}

	return config, err
}
