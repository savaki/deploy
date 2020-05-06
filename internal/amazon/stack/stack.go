// Copyright 2020 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

const (
	Env                  = "Env"
	S3Bucket             = "S3Bucket"
	S3Prefix             = "S3Prefix"
	Version              = "Version"
	CloudMapNamespaceARN = "CloudMapNamespaceARN"
)

type Operation string

func (o Operation) String() string { return string(o) }

const (
	Insert Operation = "insert"
	Update Operation = "update"
	Delete Operation = "delete"
)

type Change struct {
	Operation Operation
	Stack     Stack
}

func (c Change) String() string {
	return c.Operation.String() + " " + c.Stack.Name
}

type Stack struct {
	Name         string
	Tags         []cloudformation.Tag
	TemplateBody string
}

func Load(filename string, body io.Reader, opts ...Option) (Stack, error) {
	var (
		options = buildOptions(opts...)
		name    = makeStackName(filename)
	)

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return Stack{}, fmt.Errorf("unable to read template from file, %v: %w", filename, err)
	}

	return Stack{
		Name:         options.Prefix + options.FormatName(name),
		TemplateBody: string(data),
		Tags:         options.Tags,
	}, nil
}

func makeStackName(filename string) string {
	ext := filepath.Ext(filename)
	name := filepath.Base(filename)
	return name[0 : len(name)-len(ext)]
}

func LoadFile(filename string, opts ...Option) (Stack, error) {
	f, err := os.Open(filename)
	if err != nil {
		return Stack{}, fmt.Errorf("unable to read template, %v: %w", filename, err)
	}
	defer f.Close()

	return Load(filename, f, opts...)
}

// LoadAll stacks from the directory provided
func LoadAll(dirname string, opts ...Option) ([]Stack, error) {
	const suffix = ".template"

	var stacks []Stack
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, suffix) {
			return nil
		}

		stack, err := LoadFile(path, opts...)
		if err != nil {
			return fmt.Errorf("unable to read dir, %v: %w", dirname, err)
		}

		stacks = append(stacks, stack)

		return nil
	}
	if err := filepath.Walk(dirname, fn); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to read dir, %v: %w", dirname, err)
	}

	return stacks, nil
}
