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

package cmd

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/kadaan/log4shell-scanner/version"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultHashes = `bf4f41403280c1b115650d470f9b260a5c9042c04d9bcc2a6ca504a66379b2d6  ./apache-log4j-2.0-alpha2-bin/log4j-core-2.0-alpha2.jar
58e9f72081efff9bdaabd82e3b3efe5b1b9f1666cefe28f429ad7176a6d770ae  ./apache-log4j-2.0-beta1-bin/log4j-core-2.0-beta1.jar
ed285ad5ac6a8cf13461d6c2874fdcd3bf67002844831f66e21c2d0adda43fa4  ./apache-log4j-2.0-beta2-bin/log4j-core-2.0-beta2.jar
dbf88c623cc2ad99d82fa4c575fb105e2083465a47b84d64e2e1a63e183c274e  ./apache-log4j-2.0-beta3-bin/log4j-core-2.0-beta3.jar
a38ddff1e797adb39a08876932bc2538d771ff7db23885fb883fec526aff4fc8  ./apache-log4j-2.0-beta4-bin/log4j-core-2.0-beta4.jar
7d86841489afd1097576a649094ae1efb79b3147cd162ba019861dfad4e9573b  ./apache-log4j-2.0-beta5-bin/log4j-core-2.0-beta5.jar
4bfb0d5022dc499908da4597f3e19f9f64d3cc98ce756a2249c72179d3d75c47  ./apache-log4j-2.0-beta6-bin/log4j-core-2.0-beta6.jar
473f15c04122dad810c919b2f3484d46560fd2dd4573f6695d387195816b02a6  ./apache-log4j-2.0-beta7-bin/log4j-core-2.0-beta7.jar
b3fae4f84d4303cdbad4696554b4e8d2381ad3faf6e0c3c8d2ce60a4388caa02  ./apache-log4j-2.0-beta8-bin/log4j-core-2.0-beta8.jar
dcde6033b205433d6e9855c93740f798951fa3a3f252035a768d9f356fde806d  ./apache-log4j-2.0-beta9-bin/log4j-core-2.0-beta9.jar
85338f694c844c8b66d8a1b981bcf38627f95579209b2662182a009d849e1a4c  ./apache-log4j-2.0-bin/log4j-core-2.0.jar
db3906edad6009d1886ec1e2a198249b6d99820a3575f8ec80c6ce57f08d521a  ./apache-log4j-2.0-rc1-bin/log4j-core-2.0-rc1.jar
ec411a34fee49692f196e4dc0a905b25d0667825904862fdba153df5e53183e0  ./apache-log4j-2.0-rc2-bin/log4j-core-2.0-rc2.jar
a00a54e3fb8cb83fab38f8714f240ecc13ab9c492584aa571aec5fc71b48732d  ./apache-log4j-2.0.1-bin/log4j-core-2.0.1.jar
c584d1000591efa391386264e0d43ec35f4dbb146cad9390f73358d9c84ee78d  ./apache-log4j-2.0.2-bin/log4j-core-2.0.2.jar
8bdb662843c1f4b120fb4c25a5636008085900cdf9947b1dadb9b672ea6134dc  ./apache-log4j-2.1-bin/log4j-core-2.1.jar
c830cde8f929c35dad42cbdb6b28447df69ceffe99937bf420d32424df4d076a  ./apache-log4j-2.2-bin/log4j-core-2.2.jar
6ae3b0cb657e051f97835a6432c2b0f50a651b36b6d4af395bbe9060bb4ef4b2  ./apache-log4j-2.3-bin/log4j-core-2.3.jar
535e19bf14d8c76ec00a7e8490287ca2e2597cae2de5b8f1f65eb81ef1c2a4c6  ./apache-log4j-2.4-bin/log4j-core-2.4.jar
42de36e61d454afff5e50e6930961c85b55d681e23931efd248fd9b9b9297239  ./apache-log4j-2.4.1-bin/log4j-core-2.4.1.jar
4f53e4d52efcccdc446017426c15001bb0fe444c7a6cdc9966f8741cf210d997  ./apache-log4j-2.5-bin/log4j-core-2.5.jar
df00277045338ceaa6f70a7b8eee178710b3ba51eac28c1142ec802157492de6  ./apache-log4j-2.6-bin/log4j-core-2.6.jar
28433734bd9e3121e0a0b78238d5131837b9dbe26f1a930bc872bad44e68e44e  ./apache-log4j-2.6.1-bin/log4j-core-2.6.1.jar
cf65f0d33640f2cd0a0b06dd86a5c6353938ccb25f4ffd14116b4884181e0392  ./apache-log4j-2.6.2-bin/log4j-core-2.6.2.jar
5bb84e110d5f18cee47021a024d358227612dd6dac7b97fa781f85c6ad3ccee4  ./apache-log4j-2.7-bin/log4j-core-2.7.jar
ccf02bb919e1a44b13b366ea1b203f98772650475f2a06e9fac4b3c957a7c3fa  ./apache-log4j-2.8-bin/log4j-core-2.8.jar
815a73e20e90a413662eefe8594414684df3d5723edcd76070e1a5aee864616e  ./apache-log4j-2.8.1-bin/log4j-core-2.8.1.jar
10ef331115cbbd18b5be3f3761e046523f9c95c103484082b18e67a7c36e570c  ./apache-log4j-2.8.2-bin/log4j-core-2.8.2.jar
dc815be299f81c180aa8d2924f1b015f2c46686e866bc410e72de75f7cd41aae  ./apache-log4j-2.9.0-bin/log4j-core-2.9.0.jar
9275f5d57709e2204900d3dae2727f5932f85d3813ad31c9d351def03dd3d03d  ./apache-log4j-2.9.1-bin/log4j-core-2.9.1.jar
f35ccc9978797a895e5bee58fa8c3b7ad6d5ee55386e9e532f141ee8ed2e937d  ./apache-log4j-2.10.0-bin/log4j-core-2.10.0.jar
5256517e6237b888c65c8691f29219b6658d800c23e81d5167c4a8bbd2a0daa3  ./apache-log4j-2.11.0-bin/log4j-core-2.11.0.jar
d4485176aea67cc85f5ccc45bb66166f8bfc715ae4a695f0d870a1f8d848cc3d  ./apache-log4j-2.11.1-bin/log4j-core-2.11.1.jar
3fcc4c1f2f806acfc395144c98b8ba2a80fe1bf5e3ad3397588bbd2610a37100  ./apache-log4j-2.11.2-bin/log4j-core-2.11.2.jar
057a48fe378586b6913d29b4b10162b4b5045277f1be66b7a01fb7e30bd05ef3  ./apache-log4j-2.12.0-bin/log4j-core-2.12.0.jar
5dbd6bb2381bf54563ea15bc9fbb6d7094eaf7184e6975c50f8996f77bfc3f2c  ./apache-log4j-2.12.1-bin/log4j-core-2.12.1.jar
c39b0ea14e7766440c59e5ae5f48adee038d9b1c7a1375b376e966ca12c22cd3  ./apache-log4j-2.13.0-bin/log4j-core-2.13.0.jar
6f38a25482d82cd118c4255f25b9d78d96821d22bab498cdce9cda7a563ca992  ./apache-log4j-2.13.1-bin/log4j-core-2.13.1.jar
54962835992e303928aa909730ce3a50e311068c0960c708e82ab76701db5e6b  ./apache-log4j-2.13.2-bin/log4j-core-2.13.2.jar
e5e9b0f8d72f4e7b9022b7a83c673334d7967981191d2d98f9c57dc97b4caae1  ./apache-log4j-2.13.3-bin/log4j-core-2.13.3.jar
68d793940c28ddff6670be703690dfdf9e77315970c42c4af40ca7261a8570fa  ./apache-log4j-2.14.0-bin/log4j-core-2.14.0.jar
9da0f5ca7c8eab693d090ae759275b9db4ca5acdbcfe4a63d3871e0b17367463  ./apache-log4j-2.14.1-bin/log4j-core-2.14.1.jar
006fc6623fbb961084243cfc327c885f3c57f2eba8ee05fbc4e93e5358778c85  ./log4j-2.0-alpha1/log4j-core-2.0-alpha1.jar`
)

var (
	rootCmd = &cobra.Command{
		Use:   "log4shell-scanner [flags]",
		Short: "log4shell-scanner recursively scans a filesystem looking for affected jars.",
		Long: `log4shell-scanner recursively scans a filesystem looking for jars with a has that 
indicates they can be exploited.  The scanner with also search within jar, zip, and gzip archive.`,
		Example:               `log4shell-scanner --root=.`,
		PreRunE:               pre,
		RunE:                  run,
		DisableFlagsInUseLine: true,
	}
	roots         []string
	hashesFile    string
	printVersion  bool
	verbosity     int
	classes       []string
	includeGlobs  []string
	jars          []string
	classMatchers []string
	jarMatchers   []jarFilenameMatcher
)

func init() {
	workingDir, _ := os.Getwd()
	rootCmd.SetVersionTemplate(version.Print())
	rootCmd.Flags().StringSliceVarP(&roots, "root", "r", []string{workingDir}, "Root directory to scan")
	_ = rootCmd.MarkFlagDirname("root")
	rootCmd.Flags().StringVar(&hashesFile, "hashes", "", "SHA256 hashes of jars to match")
	_ = rootCmd.MarkFlagFilename("hashes")
	rootCmd.Flags().StringSliceVar(&jars, "jars", []string{"log4j-core-/2.0-beta9/2.15.0"}, "Jar name and semver range to match")
	rootCmd.Flags().StringSliceVar(&classes, "classes", []string{"JndiLookup"}, "Classes to match")
	rootCmd.Flags().StringSliceVar(&includeGlobs, "include-globs", []string{"**/**"}, "Globs that indicate which path to include in the scan")
	rootCmd.Flags().CountVarP(&verbosity, "verbose", "v", "Verbose logging")
	rootCmd.Flags().BoolVar(&printVersion, "version", false, "Print version")
}

type jarFilenameMatcher struct {
	name      string
	minSemver *semver.Version
	maxSemver *semver.Version
}

func (m *jarFilenameMatcher) isMatch(filename string) (bool, error) {
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

func pre(_ *cobra.Command, _ []string) error {
	if printVersion {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", version.Print())
		os.Exit(0)
	}

	for _, c := range classes {
		if strings.HasSuffix(c, ".class") {
			classMatchers = append(classMatchers, fmt.Sprintf("%s.class", c))
		} else {
			classMatchers = append(classMatchers, c)
		}
	}

	for _, j := range jars {
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
			jarMatchers = append(jarMatchers, jarFilenameMatcher{name: name, minSemver: minSemver, maxSemver: maxSemver})
		}
	}
	return nil
}

func run(_ *cobra.Command, _ []string) error {
	for _, g := range includeGlobs {
		if !doublestar.ValidatePathPattern(g) {
			return fmt.Errorf("invalid include glob: %s", g)
		}
	}

	hashes, err := getHashes()
	if err != nil {
		return fmt.Errorf("failed to load hashes: %v", err)
	}

	total := newZero()
	hits := newHitsSet()
	for _, root := range roots {
		err = WalkDir(root, func(path string, d DirEntryEx, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.DirEntry().Name() == ".git" {
					return fs.SkipDir
				}
				return nil
			}
			if !isIncluded(path) {
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
			if _, seen := (*hits)[filePath]; seen {
				return nil
			}
			*total += 1
			scanHits, scanTotal, err := scan(hashes, fileId, filePath)
			if err != nil {
				return err
			}
			*total += scanTotal
			if len(scanHits) > 0 {
				for k, v := range scanHits {
					if _, ok := (*hits)[k]; ok {
						for m, z := range v {
							(*hits)[k][m] = z
						}
					} else {
						(*hits)[k] = v
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("Total Files Scanned: %d\n", *total)
	fmt.Printf("Total Affected Files: %d\n", len(*hits))
	contentMatches := 0
	hashMatches := 0
	nameMatches := 0
	classMatches := 0
	if len(*hits) > 0 {
		for _, matchTypes := range *hits {
			for matchType := range matchTypes {
				if matchType == "CONTENT" {
					contentMatches++
				} else if matchType == "HASH" {
					hashMatches++
				} else if matchType == "NAME/VERSION" {
					nameMatches++
				} else if matchType == "CLASS" {
					classMatches++
				}
			}
		}
	}
	fmt.Printf("    Hash Matches: %d\n", hashMatches)
	fmt.Printf("    Class Matches: %d\n", classMatches)
	fmt.Printf("    Content Matches: %d\n", contentMatches)
	fmt.Printf("    Name/Version Matches: %d\n", nameMatches)
	fmt.Println("Affected Files: ")
	if len(*hits) > 0 {
		for hit, matchTypes := range *hits {
			fmt.Printf("    %s%s\n", printMatchType(matchTypes), hit)
		}
	} else {
		fmt.Println("    NONE")
	}
	if len(*hits) > 0 {
		os.Exit(2)
	}
	return nil
}

func isIncluded(path string) bool {
	includeGlobMatch := false
	for _, g := range includeGlobs {
		if ok, _ := doublestar.PathMatch(g, path); ok {
			includeGlobMatch = true
			break
		}
	}
	return includeGlobMatch
}

func getHashes() (map[string]struct{}, error) {
	hashes := map[string]struct{}{}
	var scn *bufio.Scanner
	if len(hashesFile) == 0 {
		scn = bufio.NewScanner(strings.NewReader(DefaultHashes))
	} else {
		file, err := os.Open(hashesFile)
		if err != nil {
			log.Fatal(err)
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		scn = bufio.NewScanner(file)
	}

	for scn.Scan() {
		if strings.HasPrefix(scn.Text(), "#") {
			continue
		}
		parts := strings.SplitN(scn.Text(), " ", 2)
		hashes[strings.TrimSpace(parts[0])] = struct{}{}
	}
	if err := scn.Err(); err != nil {
		return nil, err
	}
	return hashes, nil
}

type ContentReader interface {
	GetFiles() []*zip.File
	Filename() string
	GetReader() io.Reader
}

type ZipContentReader struct {
	zipReader *zip.Reader
	reader    io.Reader
	filename  string
}

func (r ZipContentReader) GetFiles() []*zip.File {
	return r.zipReader.File
}

func (r ZipContentReader) Filename() string {
	return r.filename
}

func (r ZipContentReader) GetReader() io.Reader {
	return r.reader
}

type ZipReader struct {
	zipReader *zip.ReadCloser
	reader    io.Reader
	filename  string
}

func (r ZipReader) GetFiles() []*zip.File {
	return r.zipReader.File
}

func (r ZipReader) Filename() string {
	return r.filename
}

func (r ZipReader) GetReader() io.Reader {
	return r.reader
}

func scan(hashes map[string]struct{}, id string, source interface{}) (map[string]map[string]struct{}, int, error) {
	total := 0
	hits := map[string]map[string]struct{}{}
	var err error
	var reader ContentReader
	fileId := id
	if zfile, ok := source.(*zip.File); ok {
		fileId = fmt.Sprintf("%s @ %s", fileId, zfile.Name)
		if zfile.FileInfo().IsDir() {
			if verbosity > 1 {
				fmt.Printf("### %s\n", fileId)
			}
			return hits, total, nil
		}
		total += 1
		classMatch := ""
		basename := filepath.Base(zfile.Name)
		for _, classMatcher := range classMatchers {
			match := basename == classMatcher
			if match {
				classMatch = basename
				break
			}
		}
		if len(classMatch) > 0 {
			fileId = fmt.Sprintf("%s # %s", fileId, classMatch)
			fmt.Printf("+++ %s\n", fileId)
			return map[string]map[string]struct{}{fileId: {"CLASS": struct{}{}}}, total, err
		}
		if strings.HasSuffix(zfile.Name, ".jar") {
			rc, err := zfile.Open()
			if err != nil {
				if err.Error() != "zip: not a valid zip file" {
					return hits, total, fmt.Errorf("failed to open %s: %v", fileId, err)
				} else {
					return hits, total, nil
				}
			}
			defer func(rc io.ReadCloser) {
				_ = rc.Close()
			}(rc)
			in := rc.(io.Reader)
			if _, ok := in.(io.ReaderAt); !ok {
				buffer, err := ioutil.ReadAll(in)
				if err != nil {
					return hits, total, fmt.Errorf("failed to open %s: %v", fileId, err)
				}

				in = bytes.NewReader(buffer)
			}
			zr, err := zip.NewReader(in.(io.ReaderAt), int64(zfile.UncompressedSize64))
			if err != nil {
				return hits, total, fmt.Errorf("failed to create zip reader %s: %v", fileId, err)
			}
			r, err := zfile.OpenRaw()
			if err != nil {
				return hits, total, fmt.Errorf("failed to open raw %s: %v", fileId, err)
			}
			defer func(r io.Reader) {
				_ = rc.Close()
			}(r)
			reader = ZipContentReader{zr, r, zfile.Name}
		} else {
			if verbosity > 1 {
				fmt.Printf("### %s\n", fileId)
			}
			return hits, total, nil
		}
	} else if filename, ok := source.(string); ok {
		total += 1
		if strings.HasSuffix(filename, ".jar") {
			rc, err := zip.OpenReader(filename)
			if err != nil {
				if err.Error() != "zip: not a valid zip file" {
					return hits, total, fmt.Errorf("failed to open %s: %v", fileId, err)
				} else {
					return hits, total, nil
				}
			}
			defer func(z *zip.ReadCloser) {
				_ = z.Close()
			}(rc)
			f, err := os.Open(filename)
			if err != nil {
				return hits, total, fmt.Errorf("failed to open file %s: %v", fileId, err)
			}
			defer func(f *os.File) {
				_ = f.Close()
			}(f)
			reader = ZipReader{rc, f, filename}
		} else {
			if verbosity > 1 {
				fmt.Printf("### %s\n", fileId)
			}
			return hits, total, nil
		}
	}
	contentMatch := false
	nameMatch, err := doesFilenameMatch(reader.Filename())
	if err != nil {
		return hits, total, fmt.Errorf("failed to check for filename match %s: %v", fileId, err)
	}
	hashMatch, err := checkHash(hashes, reader.GetReader())
	if err != nil {
		return hits, total, fmt.Errorf("failed to check for hash match %s: %v", fileId, err)
	}
	for _, f := range reader.GetFiles() {
		total += 1
		scanHits, scanTotal, err := scan(hashes, fileId, f)
		if err != nil {
			return hits, total, err
		}
		total += scanTotal
		if len(scanHits) > 0 {
			contentMatch = true
			for k, v := range scanHits {
				if _, ok := hits[k]; ok {
					for m, z := range v {
						hits[k][m] = z
					}
				} else {
					hits[k] = v
				}
			}
		}
	}
	matchTypes := map[string]struct{}{}
	if nameMatch {
		matchTypes["NAME/VERSION"] = struct{}{}
	}
	if hashMatch {
		matchTypes["HASH"] = struct{}{}
	}
	if contentMatch {
		matchTypes["CONTENT"] = struct{}{}
	}

	if len(matchTypes) > 0 {
		total += 1
		hits[fileId] = matchTypes
		fmt.Printf("+++ %s%s\n", printMatchType(matchTypes), fileId)
	} else if verbosity > 0 {
		fmt.Printf("--- %s\n", fileId)
	}
	return hits, total, nil
}

func printMatchType(matchTypes map[string]struct{}) string {
	keys := make([]string, 0, len(matchTypes))
	for k := range matchTypes {
		keys = append(keys, k)
	}
	return "(" + strings.Join(keys, " ") + ") "
}

func doesFilenameMatch(filename string) (bool, error) {
	if len(jarMatchers) > 0 {
		basename := filepath.Base(filename)
		for _, m := range jarMatchers {
			match, err := m.isMatch(basename)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
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

func checkHash(hashes map[string]struct{}, reader io.Reader) (bool, error) {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return false, err
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	if _, affected := hashes[hash]; affected {
		return true, nil
	}
	return false, nil
}

func WalkDir(root string, fn WalkDirFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		entry := fs.FileInfoToDirEntry(info)
		err = walkDir(root, &statDirEntryEx{&statDirEntry{entry}, root, nil, nil}, fn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

func walkDir(path string, d DirEntryEx, walkDirFn WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			err = nil
		}
		return err
	}

	var targetPath *string
	dirToRead := path
	if d.IsSymLink() {
		symLinkTargetPath, err := d.SymLinkTargetPath()
		if err := walkDirFn(*symLinkTargetPath, d, err); err != nil || !d.IsDir() {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
		targetPath = symLinkTargetPath
	}
	dirs, err := readDir(dirToRead, targetPath)
	if err != nil {
		err = walkDirFn(path, d, err)
		if err != nil {
			return err
		}
	}
	for _, d1 := range dirs {
		path1 := filepath.Join(path, d1.DirEntry().Name())
		if err := walkDir(path1, d1, walkDirFn); err != nil {
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

type WalkDirFunc func(path string, d DirEntryEx, err error) error

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

func newZero() *int {
	i := 0
	return &i
}

func newHitsSet() *map[string]map[string]struct{} {
	a := map[string]map[string]struct{}{}
	return &a
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
