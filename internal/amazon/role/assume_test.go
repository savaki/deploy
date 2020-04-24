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

package role

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"os"
	"testing"
)

func TestAssume(t *testing.T) {
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

	roleARN := "arn:aws:iam::292985526836:role/test"
	got, err := Assume(config, roleARN, "go-test")
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}

	ctx := context.Background()
	whoAmI, err := sts.New(got).GetCallerIdentityRequest(&sts.GetCallerIdentityInput{}).Send(ctx)
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}

	if got, want := *whoAmI.Arn, "arn:aws:sts::292985526836:assumed-role/test/go-test"; got != want {
		t.Fatalf("got %v; want %v", got, want)
	}
}
