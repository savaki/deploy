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
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func TestLoadAll(t *testing.T) {
	var (
		accessKeyID     = os.Getenv("AWS_ACCESS_KEY_ID")
		secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	)

	if accessKeyID == "" || secretAccessKey == "" {
		t.SkipNow()
	}

	config, err := external.LoadDefaultAWSConfig()
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []Option{
		//WithDryRun(true),
		WithPrefix("testing"),
	}
	manager := New(cloudformation.New(config), opts...)

	for _, dir := range []string{"testdata/a", "testdata/b", "testdata/c"} {
		log.Printf("--")
		summaries, err := manager.List(ctx)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		stacks, err := LoadAll(dir, opts...)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		changes := CalculateChanges(summaries, stacks)
		fmt.Println()
		fmt.Println(dir, changes)
		err = manager.Apply(ctx, changes...)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
	}
}
