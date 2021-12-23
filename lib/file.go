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
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/h2non/filetype"
	"io"
	"os"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func GetContentReaderFromFile(filename string, globMatcher GlobMatcher) (ContentReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileReader, err := NewContentFileReader(filename, stat.Size(), NewUnbufferedReadCloser(f))
	if err != nil {
		return nil, err
	}
	reader, err := GetContentReader(fileReader, globMatcher)
	if err != nil || reader == nil {
		_ = f.Close()
	}
	return reader, err
}

func GetContentReader(reader ContentFileReader, globMatcher GlobMatcher) (ContentReader, error) {
	kind, _ := filetype.Match(reader.Header())
	switch kind.Extension {
	case "tar":
		tarReader := tar.NewReader(reader)
		_, err := tarReader.Next()
		if err != nil {
			return nil, fmt.Errorf("unable to open tar file: %v", err)
		}
		return NewTarReader(reader.Filename(), tarReader, reader, globMatcher), nil
	case "gz":
		uncompressedStream, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("unable to open gzip file: %v", err)
		}
		bufferedReader := NewBufferedReadCloser(uncompressedStream)
		contentFileReader, err := NewContentFileReader(reader.Filename(), -1, bufferedReader)
		if err != nil {
			return nil, fmt.Errorf("unable to create content reader: %v", err)
		}
		return GetContentReader(contentFileReader, globMatcher)
	case "zip":
		var size int64
		var randomAccessReader io.ReaderAt
		var closer io.Closer
		if _, ok := reader.(io.ReaderAt); !ok || reader.Size() < 0 {
			buffer := bufferPool.Get().(*bytes.Buffer)
			buffer.Reset()
			closer = Closer(func() error {
				bufferPool.Put(buffer)
				return nil
			})
			_, err := buffer.ReadFrom(reader)
			if err != nil {
				_ = closer.Close()
				return nil, fmt.Errorf("unable to buffer data zip stream: %v", err)
			}
			_ = reader.Close()
			randomAccessReader = bytes.NewReader(buffer.Bytes())
			size = int64(buffer.Len())
		} else {
			randomAccessReader = reader.(io.ReaderAt)
			size = reader.Size()
		}
		r, err := zip.NewReader(randomAccessReader, size)
		if err != nil {
			_ = closer.Close()
			return nil, fmt.Errorf("unable to open zip file: %v", err)
		}
		return NewZipReader(reader.Filename(), r, reader, globMatcher, closer), nil
	}
	return nil, nil
}
