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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/rakyll/statik/fs"
	"github.com/savaki/fairy/internal/amazon/stack"
	_ "github.com/savaki/fairy/resources"
)

// Bootstrap ensures resources required by the deployment fairy exist
func Bootstrap(ctx context.Context, config Config) error {
	fileSystem, err := fs.New()
	if err != nil {
		return fmt.Errorf("unable to load statik filesystem: %w", err)
	}

	const filename = "/bootstrap.template"
	f, err := fileSystem.Open(filename)
	if err != nil {
		return fmt.Errorf("bootstrap failed: unable to open file, %v: %w", filename, err)
	}
	defer f.Close()

	s, err := stack.Load(filename, f, stack.WithPrefix("fairy"))
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	manager := stack.New(cloudformation.New(config.Target), stack.WithPrefix(config.Env))
	if err := manager.Upsert(ctx, s); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	exports, err := manager.Exports(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	for _, e := range exports {
		k, v := aws.StringValue(e.Name), aws.StringValue(e.Value)
		switch k {
		case "fairy-bootstrap-AssetBucket":
			config.Parameters[stack.S3Bucket] = v
		}
	}

	return nil
}
