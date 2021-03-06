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
	"path/filepath"
	"strings"
)

type JarScanner struct {
	jarNameMatcher JarNameMatcher
	jarHashMatcher HashMatcher
}

func NewJarScanner(jarNameMatcher JarNameMatcher, jarHashMatcher HashMatcher) JarScanner {
	return JarScanner{
		jarNameMatcher: jarNameMatcher,
		jarHashMatcher: jarHashMatcher,
	}
}

func (s JarScanner) Scan(contentReader ContentReader) ([]MatchType, error) {
	if strings.HasSuffix(contentReader.Filename(), ".jar") {
		basename := filepath.Base(contentReader.Filename())
		jarNameMatch, err := s.jarNameMatcher.IsMatch(basename)
		if err != nil {
			return []MatchType{}, fmt.Errorf("failed to check jar name/version: %v", err)
		}
		hash, err := contentReader.Hash()
		if err != nil {
			return []MatchType{}, fmt.Errorf("failed to get hash: %v", err)
		}
		jarHashMatch := s.jarHashMatcher.IsHashMatch(hash)
		if err != nil {
			return []MatchType{}, fmt.Errorf("failed to chec jar hash: %v", err)
		}
		if jarNameMatch && jarHashMatch {
			return []MatchType{JarName, JarHash}, nil
		} else if jarNameMatch {
			return []MatchType{JarName}, nil
		} else if jarHashMatch {
			return []MatchType{JarHash}, nil
		}
	}
	return []MatchType{}, nil
}
