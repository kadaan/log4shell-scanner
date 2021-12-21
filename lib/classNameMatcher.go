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

type ClassNameMatcher interface {
	IsMatch(name string) (bool, error)
}

type classNameMatcher struct {
	patterns map[string]struct{}
}

func NewClassNameMatcher(patterns []string) ClassNameMatcher {
	r := map[string]struct{}{}
	for _, p := range patterns {
		if !strings.HasSuffix(p, ".class") {
			p = fmt.Sprintf("%s.class", p)
		}
		r[p] = struct{}{}
	}
	return &classNameMatcher{patterns: r}
}

func (m *classNameMatcher) IsMatch(name string) (bool, error) {
	for p := range m.patterns {
		match, err := filepath.Match(p, name)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
