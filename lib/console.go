// Copyright © 2021 Joel Baranick <jbaranick@gmail.com>
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"fmt"
	"github.com/aquilax/truncate"
	"github.com/jwalton/gchalk"
	"time"
)

const (
	ellipsis            = "..."
	resetLine           = "\r\033[K"
	maxWidth            = 80
	minProgressUpdateMs = 100
)

type Console interface {
	Matched(message string)
	NotMatched(message string)
	Error(message string)
	Skipped(message string)
}

type console struct {
	verbosity  int
	count      int
	lastUpdate time.Time
}

func NewConsole(verbosity int) Console {
	return &console{verbosity, 0, time.Now()}
}

func (c *console) Error(message string) {
	c.count += 1
	c.println(gchalk.Yellow("!!!"), message)
}

func (c *console) Matched(message string) {
	c.count += 1
	c.println(gchalk.Red("+++"), message)
}

func (c *console) NotMatched(message string) {
	c.count += 1
	if c.verbosity > 1 {
		c.println(gchalk.Green("---"), message)
	} else {
		c.print(gchalk.Green("---"), message)
	}
}

func (c *console) Skipped(message string) {
	c.count += 1
	if c.verbosity > 0 {
		c.println(gchalk.Grey("###"), message)
	} else {
		c.print(gchalk.Grey("###"), message)
	}
}

func (c *console) print(symbol string, message string) {
	now := time.Now()
	if now.Sub(c.lastUpdate).Milliseconds() > minProgressUpdateMs {
		fmt.Printf("%s%s %-12d %s %s", resetLine, gchalk.Bold("Scanned:"), c.count, symbol, gchalk.Grey(c.truncate(maxWidth, message)))
		c.lastUpdate = now
	}
}

func (c *console) println(symbol string, message string) {
	fmt.Printf("%s%s %s\n", resetLine, symbol, message)
}

func (c *console) truncate(maxWidth int, message string) string {
	return truncate.Truncate(message, maxWidth, ellipsis, truncate.PositionMiddle)
}
