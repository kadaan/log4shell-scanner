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
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/kadaan/log4shell-scanner/lib"
	"github.com/kadaan/log4shell-scanner/version"
	"github.com/spf13/cobra"
	"os"
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
	roots           []string
	jarHashesFile   string
	classHashesFile string
	printVersion    bool
	verbosity       int
	classes         []string
	includeGlobs    []string
	jars            []string
	scanner         lib.Scanner
)

func init() {
	workingDir, _ := os.Getwd()
	rootCmd.SetVersionTemplate(version.Print())
	rootCmd.Flags().StringSliceVarP(&roots, "root", "r", []string{workingDir}, "Root directory to scan")
	_ = rootCmd.MarkFlagDirname("root")
	rootCmd.Flags().StringSliceVar(&jars, "jars", []string{"log4j-core-/2.0-beta9/2.16.0"}, "Jar name and semver range to match")
	rootCmd.Flags().StringVar(&jarHashesFile, "jar-hashes", "", "SHA256 hashes of jars to match")
	_ = rootCmd.MarkFlagFilename("jar-hashes")
	rootCmd.Flags().StringSliceVar(&classes, "classes", []string{"JndiLookup"}, "Classes to match")
	rootCmd.Flags().StringVar(&classHashesFile, "class-hashes", "", "SHA256 hashes of classes to match")
	_ = rootCmd.MarkFlagFilename("class-hashes")
	rootCmd.Flags().StringSliceVar(&includeGlobs, "include-globs", []string{"**/**"}, "Globs that indicate which path to include in the scan")
	rootCmd.Flags().CountVarP(&verbosity, "verbose", "v", "Verbose logging")
	rootCmd.Flags().BoolVar(&printVersion, "version", false, "Print version")
}

func pre(_ *cobra.Command, _ []string) error {
	if printVersion {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", version.Print())
		os.Exit(0)
	}

	classNameMatcher := lib.NewClassNameMatcher(classes)
	classHashMatcher, err := lib.NewHashMatcherFromFile(classHashesFile, lib.DefaultClassHashes)
	if err != nil {
		return fmt.Errorf("failed to load class hashes: %v", err)
	}
	classScanner := lib.NewClassScanner(classNameMatcher, classHashMatcher)

	jarNameMatcher := lib.NewJarNameMatcher()
	err = jarNameMatcher.AddMatchers(jars...)
	if err != nil {
		return fmt.Errorf("failed to load jar names: %v", err)
	}
	jarHashMatcher, err := lib.NewHashMatcherFromFile(jarHashesFile, lib.DefaultJarHashes)
	if err != nil {
		return fmt.Errorf("failed to load jar hashes: %v", err)
	}
	jarScanner := lib.NewJarScanner(jarNameMatcher, jarHashMatcher)

	scanner = lib.NewScanner(classScanner, jarScanner, includeGlobs, verbosity)
	return nil
}

func run(_ *cobra.Command, _ []string) error {
	for _, g := range includeGlobs {
		if !doublestar.ValidatePathPattern(g) {
			return fmt.Errorf("invalid include glob: %s", g)
		}
	}

	result, err := scanner.Scan(roots...)
	if err != nil {
		return err
	}
	fmt.Printf("Total Files Scanned: %d\n", result.GetTotalFilesScanned())
	fmt.Printf("Total Matched Files: %d\n", result.GetTotalFilesMatched())
	fmt.Printf("    Class Name Matches: %d\n", result.GetMatchCountByType(lib.ClassName))
	fmt.Printf("    Class Hash Matches: %d\n", result.GetMatchCountByType(lib.ClassHash))
	fmt.Printf("    Jar Name Matches: %d\n", result.GetMatchCountByType(lib.JarName))
	fmt.Printf("    Jar Hash Matches: %d\n", result.GetMatchCountByType(lib.JarHash))
	fmt.Printf("    Content Matches: %d\n", result.GetMatchCountByType(lib.Content))
	fmt.Println("Affected Files: ")
	if result.GetTotalFilesMatched() > 0 {
		for _, m := range result.GetMatches() {
			fmt.Printf("    %s\n", m)
		}
		os.Exit(2)
	} else {
		fmt.Println("    NONE")
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
