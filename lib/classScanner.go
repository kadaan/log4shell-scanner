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
	"io"
	"path/filepath"
	"strings"
)

type ClassScanner struct {
	classNameMatcher ClassNameMatcher
	classHashMatcher HashMatcher
}

func NewClassScanner(classNameMatcher ClassNameMatcher, classHashMatcher HashMatcher) ClassScanner {
	return ClassScanner{
		classNameMatcher: classNameMatcher,
		classHashMatcher: classHashMatcher,
	}
}

func (s ClassScanner) Scan(contentFile ContentFile) ([]MatchType, error) {
	if strings.HasSuffix(contentFile.Name(), ".class") {
		basename := filepath.Base(contentFile.Name())

		var err error
		classNameMatch := false
		classNameMatch, err = s.classNameMatcher.IsMatch(basename)
		if err != nil {
			return []MatchType{}, err
		}
		reader, err := contentFile.GetReader()
		if err != nil {
			return []MatchType{}, err
		}
		defer func(reader io.ReadCloser) {
			_ = reader.Close()
		}(reader)
		classHashMatch, err := s.classHashMatcher.IsMatch(reader)
		if err != nil {
			return []MatchType{}, err
		}
		if classNameMatch && classHashMatch {
			return []MatchType{ClassName, ClassHash}, nil
		} else if classNameMatch {
			return []MatchType{ClassName}, nil
		} else if classHashMatch {
			return []MatchType{ClassHash}, nil
		}
	}
	return []MatchType{}, nil
}
