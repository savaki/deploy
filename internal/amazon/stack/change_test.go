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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"reflect"
	"testing"
)

func TestMakeChangeSet(t *testing.T) {
	testCases := map[string]struct {
		stacks      []string
		summaries   []string
		wantInserts []string
		wantUpdates []string
		wantDeletes []string
	}{
		"nil": {},
		"insert": {
			stacks:      []string{"abc", "def"},
			wantInserts: []string{"abc", "def"},
		},
		"updates": {
			stacks:      []string{"abc", "def"},
			summaries:   []string{"abc", "def"},
			wantUpdates: []string{"abc", "def"},
		},
		"deletes": {
			summaries:   []string{"abc", "def"},
			wantDeletes: []string{"abc", "def"},
		},
		"kitchen sink": {
			stacks:      []string{"a", "b", "c", "d"},
			summaries:   []string{"c", "d", "e", "f"},
			wantInserts: []string{"a", "b"},
			wantUpdates: []string{"c", "d"},
			wantDeletes: []string{"e", "f"},
		},
	}

	for label, tc := range testCases {
		t.Run(label, func(t *testing.T) {
			changes := CalculateChanges(makeStackSummaries(tc.summaries), makeStacks(tc.stacks))
			if got, want := tc.wantInserts, stackNames(filter(changes, Insert)); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v; want %v", got, want)
			}
			if got, want := tc.wantUpdates, stackNames(filter(changes, Update)); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v; want %v", got, want)
			}
			if got, want := tc.wantDeletes, stackNames(filter(changes, Delete)); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v; want %v", got, want)
			}
		})
	}
}

func filter(changes []Change, op Operation) (got []Stack) {
	for _, change := range changes {
		if change.Operation == op {
			got = append(got, change.Stack)
		}
	}
	return
}

func stackNames(ss []Stack) []string {
	var items []string
	for _, s := range ss {
		items = append(items, s.Name)
	}
	return items
}

func makeStacks(ss []string) []Stack {
	var items []Stack
	for _, s := range ss {
		items = append(items, Stack{
			Name: s,
		})
	}
	return items
}

func makeStackSummaries(ss []string) []cloudformation.StackSummary {
	var items []cloudformation.StackSummary
	for _, s := range ss {
		items = append(items, cloudformation.StackSummary{
			StackName: aws.String(s),
		})
	}
	return items
}
