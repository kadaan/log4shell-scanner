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
	"sort"
	"strings"
)

type ScanFunc func(hashes map[string]struct{}, id string, source interface{}) (map[string]map[MatchType]struct{}, int, error)

type Scanner interface {
	Scan(roots ...string) (ScanResult, error)
}

type ScanFileResult struct {
	fileId     string
	matchTypes []string
}

func (s ScanFileResult) String() string {
	return fmt.Sprintf("(%s) %s", strings.Join(s.matchTypes, " "), s.fileId)
}

type ScanResult struct {
	matches           map[string]map[MatchType]struct{}
	totalFilesScanned int
}

func NewScanResult() ScanResult {
	return ScanResult{
		matches:           map[string]map[MatchType]struct{}{},
		totalFilesScanned: 0,
	}
}

func (s *ScanResult) GetMatches() []ScanFileResult {
	i := 0
	fileIds := make([]string, len(s.matches))
	for k := range s.matches {
		fileIds[i] = k
		i += 1
	}
	sort.SliceStable(fileIds, func(i, j int) bool {
		return fileIds[i] < fileIds[j]
	})

	results := make([]ScanFileResult, len(fileIds))
	for i, k := range fileIds {
		j := 0
		v := s.matches[k]
		matchTypes := make([]string, len(v))
		for m := range v {
			matchTypes[j] = s.getMatchTypeString(m)
			j += 1
		}
		sort.SliceStable(matchTypes, func(i, j int) bool {
			return matchTypes[i] < matchTypes[j]
		})
		results[i] = ScanFileResult{k, matchTypes}
	}
	return results
}

func (s *ScanResult) getMatchTypeString(m MatchType) string {
	switch m {
	case Content:
		return "CONTENT"
	case ClassName:
		return "CLASS_NAME"
	case ClassHash:
		return "CLASS_HASH"
	case JarName:
		return "JAR_NAME"
	case JarHash:
		return "JAR_HASH"
	}
	return "UNKNOWN"
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
		return true
	}
	return false
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
	err := walker.WalkDirs(func(fileId string, filePath string) error {
		if result.HasSeen(filePath) {
			return nil
		}
		result.IncrementTotal()
		scanResult, err := s.scan(fileId, filePath)
		if err != nil {
			return fmt.Errorf("failed to scan %s:\n%v", fileId, err)
		}
		result.Merge(scanResult)
		return nil
	}, roots...)
	return result, err
}

func (s *scanner) scan(id string, source interface{}) (ScanResult, error) {
	var err error
	var reader ContentReader
	result := NewScanResult()
	fileId := id
	if contentFile, ok := source.(ContentFile); ok {
		if contentFile.IsDir() {
			return result, nil
		}
		result.IncrementTotal()
		fileId = fmt.Sprintf("%s @ %s", fileId, contentFile.Name())
		if strings.HasSuffix(contentFile.Name(), ".class") {
			matchTypes, err := s.classScanner.Scan(contentFile)
			if err != nil {
				return result, fmt.Errorf("failed to scan class %s:\n%v", fileId, err)
			}
			if len(matchTypes) == 0 {
				return result, nil
			}
			result.AddMatch(fileId, matchTypes...)
			s.console.Info(fmt.Sprintf("+++ %s", fileId))
			return result, nil
		} else if strings.HasSuffix(contentFile.Name(), ".jar") || strings.HasSuffix(contentFile.Name(), ".zip") {
			ioReader, err := contentFile.GetReader()
			if err != nil {
				return result, fmt.Errorf("failed to get embedded zip reader %s:\n%v", fileId, err)
			}
			zipReader, err := NewEmbeddedZipReader(contentFile.Name(), contentFile.UncompressedSize(), ioReader)
			if err != nil {
				return result, fmt.Errorf("failed to open embedded zip %s:\n%v", fileId, err)
			}
			reader = zipReader
		} else {
			s.console.Debug(fmt.Sprintf("### %s", fileId))
			return result, nil
		}
	} else if filename, ok := source.(string); ok {

		result.IncrementTotal()
		contentReader, err := GetContentReaderFromFile(filename)
		if err != nil {
			return ScanResult{}, err
		}
		if contentReader == nil {
			s.console.Debug(fmt.Sprintf("### %s", fileId))
			return result, nil
		}
		reader = contentReader
	}
	defer func(reader ContentReader) {
		_ = reader.Close()
	}(reader)
	contentMatch := false
	matchTypes, err := s.jarScanner.Scan(reader)
	if err != nil {
		return result, fmt.Errorf("failed to scan %s:\n%v", fileId, err)
	}
	result.AddMatch(fileId, matchTypes...)
	files := reader.GetFiles()
	for {
		next, err := files.Next()
		if err != nil {
			return result, fmt.Errorf("failed to get next archive file from %s:\n%v", fileId, err)
		}
		if next == nil {
			break
		}
		contentScanResult, err := s.scan(fileId, next)
		if err != nil {
			return result, fmt.Errorf("failed to scan %s:\n%v", fileId, err)
		}
		if result.Merge(contentScanResult) {
			contentMatch = true
			result.AddMatch(fileId, Content)
		}
	}
	if contentMatch {
		s.console.Info(fmt.Sprintf("+++ %s", fileId))
	} else {
		s.console.Verbose(fmt.Sprintf("--- %s", fileId))
	}
	return result, nil
}
