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
	"archive/tar"
	"io"
)

type tarFile struct {
	header *tar.Header
	reader ContentFileReader
}

func NewTarFile(header *tar.Header, r *tar.Reader) (ContentFile, error) {
	contentFileReader, err := NewContentFileReader(header.Name, header.Size, NewNopUnbufferedCloser(r))
	if err != nil {
		return nil, err
	}
	return &tarFile{header: header, reader: contentFileReader}, nil
}

func (t *tarFile) Name() string {
	return t.header.Name
}

func (t *tarFile) UncompressedSize() int64 {
	return t.header.Size
}

func (t *tarFile) IsDir() bool {
	return t.header.Typeflag == tar.TypeDir
}

func (t *tarFile) Reader() ContentFileReader {
	return t.reader
}

func (t *tarFile) Close() error {
	return t.reader.Close()
}

type tarReader struct {
	reader            *tar.Reader
	contentFileReader ContentFileReader
	filename          string
	globMatcher       GlobMatcher
}

func NewTarReader(filename string, reader *tar.Reader, contentFileReader ContentFileReader, globMatcher GlobMatcher) ContentReader {
	return &tarReader{
		reader:            reader,
		contentFileReader: contentFileReader,
		filename:          filename,
		globMatcher:       globMatcher,
	}
}

func (r *tarReader) Files() FileIterable {
	return &tarReaderFileIterable{
		filename:    r.filename,
		reader:      r.reader,
		globMatcher: r.globMatcher,
	}
}

func (r *tarReader) Filename() string {
	return r.filename
}

func (r *tarReader) Hash() (string, error) {
	return r.contentFileReader.Hash()
}

func (r *tarReader) Close() error {
	return r.contentFileReader.Close()
}

type tarReaderFileIterable struct {
	filename    string
	reader      *tar.Reader
	globMatcher GlobMatcher
}

func (i *tarReaderFileIterable) Next() (interface{}, error) {
	for {
		next, err := i.reader.Next()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if next.Typeflag != tar.TypeReg || next.Size == 0 || !i.globMatcher.IsIncluded(next.Name) {
			continue
		}
		return NewTarFile(next, i.reader)
	}
}
