// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcxyz/pkg/protoutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain() error {
	f := flag.NewFlagSet("", flag.ExitOnError)

	if err := f.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// The sanity checker needs to have one file or directory
	args := f.Args()
	if got := len(args); got != 1 {
		return fmt.Errorf("expected one argument, got %d", got)
	}

	dir := args[0]

	files, err := fetchExtractedFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to fetch extracted files in dir %s: %w", dir, err)
	}
	var sanityCheckErrs error
	for _, f := range files {
		originF := strings.TrimPrefix(f, dir)
		fmt.Printf("Processing file %q\n", originF)
		data, err := os.ReadFile(f)
		if err != nil {
			sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("failed to read file from %q, %w\n", originF, err))
			continue
		}

		var resourceMapping v1alpha1.ResourceMapping
		if err := protoutil.FromYAML(data, &resourceMapping); err != nil {
			sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("failed to unmarshal object yaml from file %q to resource mapping: %w\n", originF, err))
			continue
		}
		for _, e := range resourceMapping.Contacts.Email {
			if !isValidEmail(e) {
				sanityCheckErrs = errors.Join(fmt.Errorf("email %q contained from file %q is not valid \n", e, originF))
			}
		}
		if resourceMapping.Resource.Provider == "" {
			sanityCheckErrs = errors.Join(fmt.Errorf("resource provider %q contained from file %q is not valid \n", resourceMapping.Resource.Provider, originF))
		}
	}

	return sanityCheckErrs
}

func fetchExtractedFiles(localDir string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(localDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking scratch directory %q: %v", path, err)
		}

		if entry.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}
