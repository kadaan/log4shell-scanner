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
	"github.com/bmatcuk/doublestar/v4"
)

type GlobMatcher interface {
	IsIncluded(path string) bool
}

type globMatcher struct {
	includeGlobs []string
	excludeGlobs []string
}

func NewGlobMatcher(includeGlobs []string, excludeGlobs []string) (GlobMatcher, error) {
	for _, g := range includeGlobs {
		if !doublestar.ValidatePathPattern(g) {
			return nil, fmt.Errorf("invalid include glob: %s", g)
		}
	}

	for _, g := range excludeGlobs {
		if !doublestar.ValidatePathPattern(g) {
			return nil, fmt.Errorf("invalid exclude glob: %s", g)
		}
	}

	return &globMatcher{
		includeGlobs: includeGlobs,
		excludeGlobs: excludeGlobs,
	}, nil
}

func (g globMatcher) IsIncluded(path string) bool {
	for _, g := range g.excludeGlobs {
		if ok, _ := doublestar.PathMatch(g, path); ok {
			return false
		}
	}

	include := false
	for _, g := range g.includeGlobs {
		if ok, _ := doublestar.PathMatch(g, path); ok {
			include = true
			break
		}
	}
	return include
}
