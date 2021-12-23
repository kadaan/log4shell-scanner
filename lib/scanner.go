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
	"sort"
	"strings"
)

type ScanFunc func(hashes map[string]struct{}, id string, source interface{}) (map[string]map[MatchType]struct{}, int, error)

type Scanner interface {
	Scan(roots ...string) (ScanResult, error)
}

type ScanMatch struct {
	fileId     string
	matchTypes []string
}

func (s ScanMatch) String() string {
	return fmt.Sprintf("(%s) %s", strings.Join(s.matchTypes, " "),
		gchalk.WithAnsi256(uint8(245+2*len(s.matchTypes))).Paint(s.fileId))
}

type ScanFailure struct {
	fileId   string
	messages []string
}

func (s ScanFailure) String() string {
	return fmt.Sprintf("%s\n        %s", s.fileId, gchalk.Grey(strings.Join(s.messages, "        \n")))
}

type ScanResult struct {
	matches           map[string]map[MatchType]struct{}
	failures          map[string]map[string]struct{}
	totalFilesScanned int
}

func NewScanResult() ScanResult {
	return ScanResult{
		matches:           map[string]map[MatchType]struct{}{},
		failures:          map[string]map[string]struct{}{},
		totalFilesScanned: 0,
	}
}

func (s *ScanResult) GetFailures() []ScanFailure {
	i := 0
	fileIds := make([]string, len(s.failures))
	for k := range s.failures {
		fileIds[i] = k
		i += 1
	}
	sort.SliceStable(fileIds, func(i, j int) bool {
		return fileIds[i] < fileIds[j]
	})

	results := make([]ScanFailure, len(fileIds))
	for i, k := range fileIds {
		j := 0
		v := s.failures[k]
		failures := make([]string, len(v))
		for m := range v {
			failures[j] = m
			j += 1
		}
		sort.SliceStable(failures, func(i, j int) bool {
			return failures[i] < failures[j]
		})
		results[i] = ScanFailure{k, failures}
	}
	return results
}

func (s *ScanResult) GetMatches() []ScanMatch {
	i := 0
	fileIds := make([]string, len(s.matches))
	for k := range s.matches {
		fileIds[i] = k
		i += 1
	}
	sort.SliceStable(fileIds, func(i, j int) bool {
		return fileIds[i] < fileIds[j]
	})

	results := make([]ScanMatch, len(fileIds))
	i = 0
	for _, k := range fileIds {
		v := s.matches[k]
		if _, content := v[Content]; len(v) > 1 || !content {
			j := 0
			matchTypes := make([]string, len(v))
			for m := range v {
				matchTypes[j] = s.getMatchTypeString(m)
				j += 1
			}
			sort.SliceStable(matchTypes, func(i, j int) bool {
				return matchTypes[i] < matchTypes[j]
			})
			results[i] = ScanMatch{k, matchTypes}
			i += 1
		}
	}
	return results[:i]
}

func (s *ScanResult) getMatchTypeString(m MatchType) string {
	switch m {
	case Content:
		return gchalk.Blue("CONTENT")
	case ClassName:
		return gchalk.Green("CLASS_NAME")
	case ClassHash:
		return gchalk.Red("CLASS_HASH")
	case JarName:
		return gchalk.Cyan("JAR_NAME")
	case JarHash:
		return gchalk.Yellow("JAR_HASH")
	}
	return gchalk.Grey("UNKNOWN")
}

func (s *ScanResult) GetTotalScanFailures() int {
	return len(s.failures)
}

func (s *ScanResult) GetTotalFilesScanned() int {
	return s.totalFilesScanned
}

func (s *ScanResult) GetTotalFilesMatched() int {
	return len(s.matches)
}

func (s *ScanResult) GetMatchCountByType(matchType MatchType) int {
	count := 0
	for _, v := range s.matches {
		if _, ok := v[matchType]; ok {
			count += 1
		}
	}
	return count
}

func (s *ScanResult) GetMatchesForFileId(id string) map[MatchType]struct{} {
	return s.matches[id]
}

func (s *ScanResult) IncrementTotal() {
	s.totalFilesScanned += 1
}

func (s *ScanResult) HasSeen(path string) bool {
	if _, seen := s.matches[path]; seen {
		return true
	}
	return false
}

func (s *ScanResult) Merge(result ScanResult) bool {
	hadMatches := false
	s.totalFilesScanned += result.totalFilesScanned
	if len(result.matches) > 0 {
		for k, v := range result.matches {
			if _, ok := s.matches[k]; ok {
				for m, z := range v {
					s.matches[k][m] = z
				}
			} else {
				s.matches[k] = v
			}
		}
		hadMatches = true
	}
	if len(result.failures) > 0 {
		for k, v := range result.failures {
			if _, ok := s.failures[k]; ok {
				for m, z := range v {
					s.failures[k][m] = z
				}
			} else {
				s.failures[k] = v
			}
		}
	}
	return hadMatches
}

func (s *ScanResult) AddMatch(id string, types ...MatchType) {
	for _, matchType := range types {
		m, ok := s.matches[id]
		if !ok {
			s.matches[id] = map[MatchType]struct{}{matchType: {}}
		} else {
			m[matchType] = struct{}{}
		}
	}
}

func (s *ScanResult) AddFailure(id string, err error) {
	m, ok := s.failures[id]
	if !ok {
		s.failures[id] = map[string]struct{}{err.Error(): {}}
	} else {
		m[err.Error()] = struct{}{}
	}
}

type scanner struct {
	classScanner ClassScanner
	jarScanner   JarScanner
	includeGlobs []string
	console      Console
}

func NewScanner(classScanner ClassScanner, jarScanner JarScanner, includeGlobs []string, verbosity int) Scanner {
	return &scanner{
		classScanner: classScanner,
		jarScanner:   jarScanner,
		includeGlobs: includeGlobs,
		console:      NewConsole(verbosity),
	}
}

func (s *scanner) Scan(roots ...string) (ScanResult, error) {
	result := NewScanResult()
	walker := NewWalker(s.includeGlobs)
	err := walker.WalkDirs(func(fileId string, filePath string, progress Progress) error {
		if result.HasSeen(filePath) {
			return nil
		}
		scanResult, err := s.scan(fileId, filePath, progress)
		if err != nil {
			result.AddFailure(fileId, fmt.Errorf("failed to scan: %v", err))
		}
		result.Merge(scanResult)
		return nil
	}, roots...)
	return result, err
}

func (s *scanner) scan(id string, source interface{}, progress Progress) (ScanResult, error) {
	var err error
	var reader ContentReader
	result := NewScanResult()
	fileId := id
	if contentFile, ok := source.(ContentFile); ok {
		defer func(contentFile ContentFile) {
			_ = contentFile.Close()
		}(contentFile)

		if contentFile.IsDir() {
			return result, nil
		}
		result.IncrementTotal()
		fileId = fmt.Sprintf("%s @ %s", fileId, contentFile.Name())
		if strings.HasSuffix(contentFile.Name(), ".class") {
			matchTypes, err := s.classScanner.Scan(contentFile)
			if err != nil {
				result.AddFailure(fileId, fmt.Errorf("failed to scan class: %v", err))
				s.console.Error(progress, fileId)
				return result, nil
			}
			if len(matchTypes) == 0 {
				s.console.NotMatched(progress, fileId)
				return result, nil
			}
			result.AddMatch(fileId, matchTypes...)
			s.console.Matched(progress, fileId)
			return result, nil
		} else {
			contentFileReader := contentFile.Reader()
			contentReader, err := GetContentReader(contentFileReader)
			if err != nil {
				result.AddFailure(fileId, err)
				s.console.Error(progress, fileId)
				return result, nil
			}
			if contentReader == nil {
				s.console.Skipped(progress, fileId)
				return result, nil
			}
			reader = contentReader
		}
	} else if filename, ok := source.(string); ok {
		result.IncrementTotal()
		contentReader, err := GetContentReaderFromFile(filename)
		if err != nil {
			result.AddFailure(fileId, err)
			s.console.Error(progress, fileId)
			return result, nil
		}
		if contentReader == nil {
			s.console.Skipped(progress, fileId)
			return result, nil
		}
		reader = contentReader
	}
	matchTypes, err := s.jarScanner.Scan(reader)
	if err != nil {
		result.AddFailure(fileId, err)
		s.console.Error(progress, fileId)
		return result, nil
	}
	result.AddMatch(fileId, matchTypes...)
	files := reader.Files()
	for {
		next, err := files.Next()
		if err != nil {
			result.AddFailure(fileId, fmt.Errorf("failed to get next archive file: %v", err))
			s.console.Error(progress, fileId)
			return result, nil
		}
		if next == nil {
			break
		}
		contentScanResult, err := s.scan(fileId, next, progress)
		if err != nil {
			result.AddFailure(fileId, fmt.Errorf("failed to scan: %v", err))
			s.console.Error(progress, fileId)
		}
		if result.Merge(contentScanResult) {
			result.AddMatch(fileId, Content)
		}
	}
	currentMatches := result.GetMatchesForFileId(fileId)
	_, contentMatch := currentMatches[Content]
	if len(currentMatches) > 1 || (len(currentMatches) > 0 && !contentMatch) {
		s.console.Matched(progress, fileId)
	} else {
		s.console.NotMatched(progress, fileId)
	}
	return result, nil
}
