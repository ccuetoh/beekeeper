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
	// DefaultPort is the default port for Beekeeper servers
	DefaultPort = 2020

	// DefaultScanTime is the scan time to be used by scan functions
	DefaultScanTime = time.Second * 2
)

// WatchdogSleep is the time between node pings for the watchdog
var WatchdogSleep = time.Second * 15

// Config holds the configurations for a node or a primary node.
type Config struct {
	// Name of the node. It defaults to the system's hostname.
	Name string `mapstructure:"name,omitempty"`

	// Debug toggles between verbosity for debugging.
	Debug bool `mapstructure:"debug,omitempty"`

	// Token is a passphrase used to restrict usage of the node. Must match on the receiving node.
	Token string `mapstructure:"token,omitempty"`

	// InboundPort is the port to be used for receiving connections. Defaults to 2020.
	InboundPort int `mapstructure:"inbound_port,omitempty"`

	// OutboundPort is the port assumed to be used by a remote node. It's only used to establish a connection, and
	// afterwards a port is negotiated with the remote node. Defaults to 2020.
	OutboundPort int `mapstructure:"outbound_port,omitempty"`

	// TLSCertificate is used for TLS connections between nodes. If none is given a certificate is created on the first
	// run and reused as needed.
	TLSCertificate []byte

	// TLSPrivateKey is used for TLS connections between nodes. If none is given a key is created on the first
	// run and reused as needed.
	TLSPrivateKey []byte

	// AllowExternal sets whether non-local connections should be accepted. It's heavily encouraged that a whitelist
	// and token is set with this featured turn on. Defaults to false.
	AllowExternal bool

	// Whitelist contains a list of allowed hosts. If none is provided it's understood that the whitelist is disabled.
	// A wildcard sign (*) can be used.
	Whitelist []string `mapstructure:"whitelist,omitempty"`

	// MaxMessageSize is the size limit in bytes for incoming messages. It defaults to 1.024 MB
	MaxMessageSize uint64 `mapstructure:"max_message_size,omitempty"`

	// DisableCleanup turns off the post-build cleanup
	DisableCleanup bool `mapstructure:"disable_cleanup,omitempty"`

	// DisableConnectionWatchdog disables the connection watchdog, and stops disconnection notifications.
	DisableConnectionWatchdog bool `mapstructure:"disable_connection_watchdog,omitempty"`
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
	c.DisableCleanup = false
	c.AllowExternal = false
	c.MaxMessageSize = 1 << 9 // 1.024 MB

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
