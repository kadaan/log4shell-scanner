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
	"archive/zip"
	"fmt"
)

type zipFile struct {
	file   *zip.File
	reader ContentFileReader
}

func NewZipFile(file *zip.File) (ContentFile, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("unable to open zip content %s:\n%v", file.Name, err)
	}
	contentFileReader, err := NewContentFileReader(reader)
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	return &zipFile{file: file, reader: contentFileReader}, nil
}

func (z *zipFile) Name() string {
	return z.file.Name
}

func (z *zipFile) IsDir() bool {
	return z.file.FileInfo().IsDir()
}

func (z *zipFile) UncompressedSize() int64 {
	return int64(z.file.UncompressedSize64)
}

func (z *zipFile) GetReader() ContentFileReader {
	return z.reader
}

func (z *zipFile) Close() error {
	return z.reader.Close()
}

type zipReader struct {
	reader            *zip.Reader
	contentFileReader ContentFileReader
	filename          string
}

func NewZipReader(filename string, reader *zip.Reader, contentFileReader ContentFileReader) ContentReader {
	return &zipReader{
		reader:            reader,
		contentFileReader: contentFileReader,
		filename:          filename,
	}
}

func (r *zipReader) GetFiles() FileIterable {
	return &zipReaderFileIterable{
		index: 0,
		files: r.reader.File,
	}
}

func (r *zipReader) Filename() string {
	return r.filename
}

func (r *zipReader) GetHash() (string, error) {
	return r.contentFileReader.GetHash()
}

func (r *zipReader) Close() error {
	return r.contentFileReader.Close()
}

type zipReaderFileIterable struct {
	index int
	files []*zip.File
}

func (i *zipReaderFileIterable) Next() (interface{}, error) {
	for {
		if i.index >= len(i.files)-1 {
			return nil, nil
		}
		current := i.index
		i.index += 1
		if i.files[current].FileInfo().IsDir() {
			continue
		}
		return NewZipFile(i.files[current])
	}
}
