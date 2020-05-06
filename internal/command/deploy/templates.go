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

package deploy

import (
	"context"
	"fmt"
	"github.com/savaki/fairy/internal/banner"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/savaki/fairy/internal/amazon/stack"
)

// Templates upserts all templates from ${config.Dir}/templates if it exists
func Templates(ctx context.Context, config Config) error {
	banner.Println("deploying cloudformation templates ...")

	opts := []stack.Option{
		stack.WithPrefix(config.Env + "-" + config.Project),
		stack.WithNameFormatter(func(s string) string { return "-" + s }),
		stack.WithParameters(config.Parameters),
	}

	dir := filepath.Join(config.Dir, "templates")
	stacks, err := stack.LoadAll(dir, opts...)
	if err != nil {
		return fmt.Errorf("unable to load templates from dir, %v: %w", dir, err)
	}

	manager := stack.New(cloudformation.New(config.Target), opts...)
	summaries, err := manager.List(ctx)
	if err != nil {
		return fmt.Errorf("unable to load templates from dir, %v: %w", dir, err)
	}

	changes := stack.CalculateChanges(summaries, stacks)
	if err := manager.Apply(ctx, changes...); err != nil {
		return fmt.Errorf("unable to apply templates: %w", err)
	}

	return nil
}
