// MIT License
//
// # Copyright (c) 2018 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

func (configuration *Configuration) migrateToLatestVersion() {
	log.Debugf("⚙ Migrating configuration to latest version...")
	if configuration.Version == 0 {
		configuration.migrateVersion0()
	}
	log.Debugf("⚙ Migration of configuration complete")
}

func (configuration *Configuration) migrateVersion0() {

}

func migrateTimestampFormat(timestamp string) (string, error) {
	// Check for old format and convert
	layout := "3:04PM"
	t, err := time.Parse(layout, timestamp)
	if err == nil {
		log.Debugf("⚙ Migrating old timestamp %s to %s", timestamp, t.Format("15:04"))
		return t.Format("15:04"), nil
	}

	// Already new format? Return unchanged
	layout = "15:04"
	t, err = time.Parse(layout, timestamp)
	if err == nil {
		return timestamp, nil
	}

	return "", fmt.Errorf("Invalid timestamp format: %s", timestamp)
}
