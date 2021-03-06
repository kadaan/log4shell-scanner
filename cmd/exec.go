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
	"context"
	"fmt"
	"github.com/jwalton/gchalk"
	"github.com/kadaan/log4shell-scanner/lib"
	"github.com/kadaan/log4shell-scanner/version"
	"github.com/spf13/cobra"
	"github.com/thecodeteam/goodbye"
	"os"
	"strconv"
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
	excludeGlobs    []string
	jars            []string
	scanner         lib.Scanner
)

const (
	exitCodeAnnotationKey = "exit_code"
)

func init() {
	workingDir, _ := os.Getwd()
	rootCmd.SetVersionTemplate(version.Print())
	rootCmd.Flags().StringSliceVarP(&roots, "root", "r", []string{workingDir}, "Root directory to scan (repeatable)")
	_ = rootCmd.MarkFlagDirname("root")
	rootCmd.Flags().StringSliceVar(&jars, "jars", []string{"log4j-core-/2.0-beta9/2.16.0"}, "Jar name and semver range to match (repeatable)")
	rootCmd.Flags().StringVar(&jarHashesFile, "jar-hashes", "", "File containing SHA256 hashes of jars to match")
	_ = rootCmd.MarkFlagFilename("jar-hashes")
	rootCmd.Flags().StringSliceVar(&classes, "classes", []string{"JndiLookup"}, "Classes to match (repeatable)")
	rootCmd.Flags().StringVar(&classHashesFile, "class-hashes", "", "File containing SHA256 hashes of classes to match")
	_ = rootCmd.MarkFlagFilename("class-hashes")
	rootCmd.Flags().StringSliceVar(&includeGlobs, "include-globs", []string{"**/**"}, "Globs that indicate which paths to include in the scan (repeatable)")
	rootCmd.Flags().StringSliceVar(&excludeGlobs, "exclude-globs", []string{"**/.git/**", "**/.runtime/**", "**/node_modules/**"}, "Globs that indicate which paths to exclude in the scan (repeatable)")
	rootCmd.Flags().CountVarP(&verbosity, "verbose", "v", "Verbose logging")
	rootCmd.Flags().BoolVar(&printVersion, "version", false, "Print version")
	lib.AddProfileFlags(rootCmd)
}

func pre(_ *cobra.Command, _ []string) error {
	if printVersion {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", version.Print())
		os.Exit(0)
	}

	globMatcher, err := lib.NewGlobMatcher(includeGlobs, excludeGlobs)
	if err != nil {
		return err
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

	scanner = lib.NewScanner(classScanner, jarScanner, globMatcher, verbosity)
	return nil
}

func run(cmd *cobra.Command, _ []string) error {
	err := lib.StartProfiling()
	if err != nil {
		return err
	}

	result, err := scanner.Scan(roots...)
	if err != nil {
		return err
	}
	fmt.Printf("%s\nTotal Files Scanned: %d\n", lib.ResetLine, result.GetTotalFilesScanned())
	fmt.Printf("\nTotal Matched Files: %d\n", result.GetTotalFilesMatched())
	fmt.Printf("    Content Matches: %s\n", gchalk.Blue(fmt.Sprintf("%d", result.GetMatchCountByType(lib.Content))))
	fmt.Printf("    Class Name Matches: %s\n", gchalk.Green(fmt.Sprintf("%d", result.GetMatchCountByType(lib.ClassName))))
	fmt.Printf("    Class Hash Matches: %s\n", gchalk.Red(fmt.Sprintf("%d", result.GetMatchCountByType(lib.ClassHash))))
	fmt.Printf("    Jar Name Matches: %s\n", gchalk.Cyan(fmt.Sprintf("%d", result.GetMatchCountByType(lib.JarName))))
	fmt.Printf("    Jar Hash Matches: %s\n", gchalk.Yellow(fmt.Sprintf("%d", result.GetMatchCountByType(lib.JarHash))))
	fmt.Println("\nMatched Files: ")

	exitCode := 0
	if result.GetTotalFilesMatched() > 0 {
		exitCode += 2
		for _, m := range result.GetMatches() {
			fmt.Printf("    %s\n", m)
		}
	} else {
		fmt.Println("    NONE")
	}

	fmt.Printf("\nTotal Scan Failures: %d\n", result.GetTotalScanFailures())
	fmt.Println("\nFailed Files: ")
	if result.GetTotalScanFailures() > 0 {
		exitCode += 4
		for _, m := range result.GetFailures() {
			fmt.Printf("    %s\n", m)
		}
	} else {
		fmt.Println("    NONE")
	}
	cmd.Annotations = make(map[string]string)
	cmd.Annotations[exitCodeAnnotationKey] = fmt.Sprintf("%d", exitCode)
	return nil
}

func Execute() {
	ctx := context.Background()
	defer goodbye.Exit(ctx, -1)
	goodbye.Notify(ctx)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	if exitCodeString, ok := rootCmd.Annotations[exitCodeAnnotationKey]; ok {
		exitCode, err := strconv.ParseInt(exitCodeString, 10, 8)
		if err != nil {
			goodbye.Exit(ctx, 1)
		}
		goodbye.Exit(ctx, int(exitCode))
	} else {
		goodbye.Exit(ctx, 0)
	}
}
