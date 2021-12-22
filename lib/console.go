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
	"github.com/jwalton/gchalk"
)

type Console interface {
	Matched(message string)
	NotMatched(message string)
	Error(message string)
	Skipped(message string)
}

type console struct {
	verbosity int
}

func NewConsole(verbosity int) Console {
	return &console{verbosity}
}

func (c *console) Error(message string) {
	fmt.Printf("%s %s\n", gchalk.Yellow("!!!"), message)
}

func (c *console) Matched(message string) {
	fmt.Printf("%s %s\n", gchalk.Red("+++"), message)
}

func (c *console) NotMatched(message string) {
	if c.verbosity > 1 {
		fmt.Printf("%s %s\n", gchalk.Green("---"), message)
	}
}

func (c *console) Skipped(message string) {
	if c.verbosity > 0 {
		fmt.Println(gchalk.Grey("### ", message, "\n"))
	}
}
