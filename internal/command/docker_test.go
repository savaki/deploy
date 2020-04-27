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

package command

import (
	"testing"
)

func Test_parseImage(t *testing.T) {
	s := "123456789012.dkr.ecr.us-west-2.amazonaws.com/dd-server:0.abc123"
	image, err := parseImage(s)
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}
	if got, want := image.String(), s; got != want {
		t.Fatalf("got %v; want %v", got, want)
	}
}
