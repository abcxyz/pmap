// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/protoutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
)

var _ cli.Command = (*ValidateCommand)(nil)

type ValidateCommand struct {
	cli.BaseCommand

	flagType string
	flagPath string
}

func (c *ValidateCommand) Desc() string {
	return `Given the type of YAML resources, verify YAML files that exists in the given path`
}

func (c *ValidateCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Given the type of YAML resources, verify YAML files that exists in the given path:

      pmapctl validate -type ResourceMapping -path "/path/to/file"
`
}

func (c *ValidateCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "type",
		Target:  &c.flagType,
		Example: "ResourceMapping",
		Usage:   `The type of the data stored in the YAML files`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "path",
		Target:  &c.flagPath,
		Example: "/path/to/file",
		Usage:   `The path of YAML files.`,
	})

	return set
}

func (c *ValidateCommand) Run(ctx context.Context, args []string) error {
	// TODO: make it generic to support different types.
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if c.flagType == "" {
		return fmt.Errorf("type is required")
	}
	if c.flagPath == "" {
		return fmt.Errorf("path is required")
	}
	if c.flagType != "ResourceMapping" {
		return fmt.Errorf("we only support type `ResourceMapping` now")
	}

	dir := c.flagPath
	files, err := fetchExtractedYAMLFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to fetch extracted files in dir %s: %w", dir, err)
	}
	var sanityCheckErrs error
	for _, file := range files {
		originF := strings.TrimPrefix(file, dir)
		fmt.Fprintf(c.Stdout(), "processing file %q\n", originF)
		data, err := os.ReadFile(file)
		if err != nil {
			sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("failed to read file from %q, %w", originF, err))
			continue
		}

		var resourceMapping v1alpha1.ResourceMapping
		if err = protoutil.FromYAML(data, &resourceMapping); err != nil {
			sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("file %q failed to pass the validation: failed to unmarshal object yaml to resource mapping: %w", originF, err))
			continue
		}
		for _, e := range resourceMapping.Contacts.Email {
			if !isValidEmail(e) {
				sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("file %q failed to pass the validation: email %q is not valid", originF, e))
			}
		}
		if resourceMapping.Resource.Provider == "" {
			sanityCheckErrs = errors.Join(sanityCheckErrs, fmt.Errorf("file %q failed to pass the validation: resource provider %q is not valid", originF, resourceMapping.Resource.Provider))
		}
	}

	return sanityCheckErrs
}

func fetchExtractedYAMLFiles(localDir string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(localDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walking scratch directory %q: %w", path, err)
		}

		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk the directory %s: %w", localDir, err)
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
