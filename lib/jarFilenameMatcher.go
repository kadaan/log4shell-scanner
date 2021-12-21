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
	"github.com/Masterminds/semver"
	"strings"
)

type JarNameMatcher interface {
	IsMatch(filename string) (bool, error)
	AddMatchers(matchers ...string) error
}

func NewJarNameMatcher() JarNameMatcher {
	return &jarNameMatcher{[]matcher{}}
}

type jarNameMatcher struct {
	matchers []matcher
}

func (m *jarNameMatcher) IsMatch(filename string) (bool, error) {
	for _, m := range m.matchers {
		match, err := m.isMatch(filename)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

func (m *jarNameMatcher) AddMatchers(matchers ...string) error {
	for _, j := range matchers {
		parts := strings.SplitN(j, "/", 3)
		if parts != nil {
			name := parts[0]
			if !strings.HasSuffix(name, "-") {
				name = fmt.Sprintf("%s-", name)
			}
			var err error
			var minSemver *semver.Version
			var maxSemver *semver.Version
			if len(parts) > 1 {
				if len(parts[1]) > 0 {
					minSemver, err = semver.NewVersion(parts[1])
					if err != nil {
						return fmt.Errorf("invalid minimum semantic version in %s: %v", j, err)
					}
				}
			}
			if len(parts) == 3 {
				if len(parts[2]) > 0 {
					maxSemver, err = semver.NewVersion(parts[2])
					if err != nil {
						return fmt.Errorf("invalid maximum semantic version in %s: %v", j, err)
					}
				}
			}
			m.matchers = append(m.matchers, matcher{
				name:      name,
				minSemver: minSemver,
				maxSemver: maxSemver,
			})
		}
	}
	return nil
}

type matcher struct {
	name      string
	minSemver *semver.Version
	maxSemver *semver.Version
}

func (m *matcher) isMatch(filename string) (bool, error) {
	if strings.HasPrefix(filename, m.name) {
		filename = fileNameWithoutExtension(filename)
		if m.minSemver != nil || m.maxSemver != nil {
			ver := filename[len(m.name):]
			semVersion, err := semver.NewVersion(ver)
			if err != nil {
				return false, err
			}
			if m.minSemver != nil && m.maxSemver != nil {
				if semVersion.Compare(m.minSemver) >= 0 && semVersion.Compare(m.maxSemver) <= 0 {
					return true, nil
				}
			} else if m.minSemver != nil {
				if semVersion.Compare(m.minSemver) >= 0 {
					return true, nil
				}

			} else if m.maxSemver != nil {
				if semVersion.Compare(m.maxSemver) <= 0 {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func fileNameWithoutExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}
