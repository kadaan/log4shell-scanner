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
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type HashMatcher interface {
	IsMatch(reader io.Reader) (bool, error)
	IsHashMatch(hash string) bool
}

type hashMatcher struct {
	hashes map[string]struct{}
}

func NewHashMatcherFromString(content string) (HashMatcher, error) {
	scn := bufio.NewScanner(strings.NewReader(content))
	return newHashMatcher(scn)
}

func NewHashMatcherFromFile(file string, defaults string) (HashMatcher, error) {
	if len(file) == 0 {
		return NewHashMatcherFromString(defaults)
	} else {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(f)
		scn := bufio.NewScanner(f)
		return newHashMatcher(scn)
	}
}

func newHashMatcher(scn *bufio.Scanner) (HashMatcher, error) {
	hashes := map[string]struct{}{}
	for scn.Scan() {
		if strings.HasPrefix(scn.Text(), "#") {
			continue
		}
		parts := strings.SplitN(scn.Text(), " ", 2)
		hashes[strings.TrimSpace(parts[0])] = struct{}{}
	}
	if err := scn.Err(); err != nil {
		return nil, err
	}
	return &hashMatcher{hashes: hashes}, nil
}

func (h *hashMatcher) IsMatch(reader io.Reader) (bool, error) {
	s256 := sha256.New()
	if _, err := io.Copy(s256, reader); err != nil {
		return false, err
	}
	hash := fmt.Sprintf("%x", s256.Sum(nil))
	return h.IsHashMatch(hash), nil
}

func (h *hashMatcher) IsHashMatch(hash string) bool {
	_, match := h.hashes[hash]
	return match
}
