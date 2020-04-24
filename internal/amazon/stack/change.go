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
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

var (
	deletedStatus = []cloudformation.StackStatus{
		cloudformation.StackStatusDeleteFailed,
		cloudformation.StackStatusDeleteComplete,
	}
)

func CalculateChanges(got []cloudformation.StackSummary, want []Stack) []Change {
	got = exclude(got, deletedStatus...)

	var changes []Change
loop:
	for _, w := range want {
		for _, g := range got {
			if w.Name == *g.StackName {
				changes = append(changes, Change{
					Operation: Update,
					Stack:     w,
				})
				continue loop
			}
		}
		changes = append(changes, Change{
			Operation: Insert,
			Stack:     w,
		})
	}

del:
	for _, g := range got {
		for _, w := range want {
			if *g.StackName == w.Name {
				continue del
			}
		}
		changes = append(changes, Change{
			Operation: Delete,
			Stack:     Stack{Name: *g.StackName},
		})
	}
	return changes
}

func exclude(item []cloudformation.StackSummary, statuses ...cloudformation.StackStatus) []cloudformation.StackSummary {
	var ss []cloudformation.StackSummary
	for _, item := range item {
		if containsStatus(statuses, item.StackStatus) {
			continue
		}
		ss = append(ss, item)
	}
	return ss
}

func containsStatus(ss []cloudformation.StackStatus, want cloudformation.StackStatus) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
