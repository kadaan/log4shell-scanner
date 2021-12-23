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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type WalkDirFunc func(fileId string, filePath string, p Progress) error

type walkDirFunc func(root string, path string, d DirEntryEx, p Progress, err error) error

type Walker interface {
	WalkDirs(fn WalkDirFunc, roots ...string) error
}

type walker struct {
	globMatcher GlobMatcher
	seenPaths   map[string]struct{}
}

func NewWalker(globMatcher GlobMatcher) Walker {
	return &walker{
		globMatcher: globMatcher,
		seenPaths:   map[string]struct{}{},
	}
}

func (w *walker) WalkDirs(fn WalkDirFunc, roots ...string) error {
	p := &progress{
		current: 0,
		total:   len(roots),
	}
	for _, root := range roots {
		absRoot, err := AbsolutePath(root)
		if err != nil {
			return err
		}
		root = absRoot
		info, err := os.Lstat(root)
		if err != nil {
			err = w.walkDirEx(fn, root, root, nil, p, err)
		} else {
			entry := fs.FileInfoToDirEntry(info)
			err = w.walkDir(root, root, &statDirEntryEx{
				&statDirEntry{entry},
				root,
				nil,
				nil,
			}, p, func(root string, path string, d DirEntryEx, p Progress, err error) error {
				return w.walkDirEx(fn, root, path, d, p, err)
			})
		}
		if err == filepath.SkipDir {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) walkDirEx(fn WalkDirFunc, root string, path string, d DirEntryEx, p Progress, err error) error {
	if _, seen := w.seenPaths[path]; seen {
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}
	w.seenPaths[path] = struct{}{}
	if !w.globMatcher.IsIncluded(path) {
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}
	fileId, _ := filepath.Rel(root, path)
	filePath := path
	if d.IsSymLink() {
		targetPath, err := d.SymLinkTargetPath()
		if targetPath == nil || err != nil {
			return err
		}
		filePath = *targetPath
		relTargetPath, _ := filepath.Rel(path, filePath)
		fileId = fmt.Sprintf("%s (%s)", fileId, relTargetPath)
	}
	w.seenPaths[filePath] = struct{}{}
	return fn(fileId, filePath, p)
}

func (w *walker) walkDir(root string, path string, d DirEntryEx, p Progress, walkDirFn walkDirFunc) error {
	p.Increment()
	if err := walkDirFn(root, path, d, p, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			err = nil
		}
		return err
	}

	var targetPath *string
	dirToRead := path
	if d.IsSymLink() {
		symLinkTargetPath, err := d.SymLinkTargetPath()
		if err := walkDirFn(root, *symLinkTargetPath, d, p, err); err != nil || !d.IsDir() {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
		targetPath = symLinkTargetPath
	}
	dirs, err := readDir(dirToRead, targetPath)
	p.AddToTotal(len(dirs))
	if err != nil {
		err = walkDirFn(root, path, d, p, err)
		if err != nil {
			return err
		}
	}
	for _, d1 := range dirs {
		path1 := filepath.Join(path, d1.DirEntry().Name())
		if err := w.walkDir(root, path1, d1, p, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

func readDir(dirname string, targetPath *string) ([]DirEntryEx, error) {
	dirToRead := dirname
	if targetPath != nil {
		dirToRead = *targetPath
	}
	f, err := os.Open(dirToRead)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			err = nil
		}
		return nil, err
	}
	dirs, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	dirsEx := make([]DirEntryEx, len(dirs))
	for i, d := range dirs {
		targetPath1 := targetPath
		if targetPath1 != nil {
			fullTargetPath := filepath.Join(*targetPath1, d.Name())
			targetPath1 = &fullTargetPath
		}
		dirsEx[i] = &statDirEntryEx{d, filepath.Join(dirname, d.Name()), targetPath1, nil}
	}
	sort.Slice(dirsEx, func(i, j int) bool { return dirsEx[i].DirEntry().Name() < dirsEx[j].DirEntry().Name() })
	return dirsEx, nil
}

type Progress interface {
	Current() int
	Total() int
	AddToTotal(i int)
	Increment()
}

type progress struct {
	current int
	total   int
}

func (p *progress) Current() int {
	return p.current
}

func (p *progress) Total() int {
	return p.total
}

func (p *progress) AddToTotal(i int) {
	p.total += i
}

func (p *progress) Increment() {
	p.current += 1
}

type DirEntryEx interface {
	DirEntry() fs.DirEntry
	IsSymLink() bool
	IsDir() bool
	SymLinkTargetPath() (*string, error)
	SymLinkTargetEntry() fs.DirEntry
}

type statDirEntryEx struct {
	entry       fs.DirEntry
	path        string
	targetPath  *string
	targetEntry fs.DirEntry
}

func (d *statDirEntryEx) DirEntry() fs.DirEntry { return d.entry }
func (d *statDirEntryEx) IsSymLink() bool {
	return d.targetPath != nil || d.entry.Type()&os.ModeSymlink == os.ModeSymlink
}
func (d *statDirEntryEx) IsDir() bool {
	if d.IsSymLink() {
		return d.SymLinkTargetEntry().IsDir()
	}
	return d.entry.IsDir()
}
func (d *statDirEntryEx) SymLinkTargetPath() (*string, error) {
	if !d.IsSymLink() {
		return nil, nil
	}
	if d.targetPath == nil {
		finalPath, err := filepath.EvalSymlinks(d.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || err.Error() == "EvalSymlinks: too many links" {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to eval symlink %s: %v", d.path, err)
		}
		d.targetPath = &finalPath
	}
	return d.targetPath, nil
}
func (d *statDirEntryEx) SymLinkTargetEntry() fs.DirEntry {
	if d.IsSymLink() {
		if d.targetEntry != nil {
			return d.targetEntry
		}
		targetPath, err := d.SymLinkTargetPath()
		if targetPath != nil && err == nil {
			lstat, err := os.Lstat(*targetPath)
			if err == nil {
				entry := fs.FileInfoToDirEntry(lstat)
				d.targetEntry = &statDirEntry{entry}
			} else {
				d.targetEntry = d.entry
			}
		} else {
			d.targetEntry = d.entry
		}
		return d.targetEntry
	}
	return nil
}

type statDirEntry struct {
	info fs.DirEntry
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info.Info() }
