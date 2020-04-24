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
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

type Options struct {
	DryRun     bool
	Parameters map[string]string
	FormatName func(string) string
	Prefix     string
	Tags       []cloudformation.Tag
}

type Option func(o *Options)

func defaultNameFormatter(s string) string {
	return s
}

func WithDryRun(dryRun bool) Option {
	return func(o *Options) {
		o.DryRun = dryRun
	}
}

func WithNameFormatter(fn func(string) string) Option {
	return func(o *Options) {
		if fn == nil {
			o.FormatName = defaultNameFormatter
		} else {
			o.FormatName = fn
		}
	}
}

func WithParameters(exports map[string]string) Option {
	return func(o *Options) {
		for k, v := range exports {
			if o.Parameters == nil {
				o.Parameters = map[string]string{}
			}
			o.Parameters[k] = v
		}
	}
}

func WithPrefix(prefix string) Option {
	prefix = strings.TrimRight(prefix, "-") + "-"

	return func(o *Options) {
		o.Prefix = prefix
	}
}

func WithTags(tags ...cloudformation.Tag) Option {
	return func(o *Options) {
		o.Tags = append(o.Tags, tags...)
	}
}

func buildOptions(opts ...Option) Options {
	options := Options{
		FormatName: defaultNameFormatter,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return options
}
