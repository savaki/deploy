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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/fatih/color"
)

func isErrShown(err error) bool {
	v, ok := err.(awserr.Error)
	if !ok {
		return true
	}

	if code := v.Code(); code == "RequestCanceled" || code == errValidationError {
		return false
	}

	return true
}

func observeEvents(ctx context.Context, api cloudformationiface.ClientAPI, stackName string) {
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	now := time.Now().Add(-12 * time.Second)
	seen := map[string]struct{}{} // keep track of events we've seen

outer:
	for iter := 0; true; iter++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		input := cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackName),
		}

		var token *string
		for {
			input.NextToken = token

			req := api.DescribeStackEventsRequest(&input)
			resp, err := req.Send(ctx)
			if err != nil {
				if isErrShown(err) {
					fmt.Printf("describe stack events failed for stack, %v - %v\n", stackName, err)
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(12 * time.Second):
					continue outer
				}
			}

			var events []cloudformation.StackEvent
			for _, event := range resp.StackEvents {
				if event.Timestamp.Before(now) {
					break
				}

				if _, ok := seen[*event.EventId]; ok {
					continue
				}
				seen[*event.EventId] = struct{}{}

				events = append(events, event)
			}

			for i := len(events) - 1; i >= 0; i-- {
				event := events[i]
				text := fmt.Sprintf("%s %-25s %-35s %-35s %s\n",
					event.Timestamp.In(time.Local).Format("2006/01/02 15:04:05"),
					aws.StringValue(event.LogicalResourceId),
					aws.StringValue(event.ResourceType),
					event.ResourceStatus,
					aws.StringValue(event.ResourceStatusReason),
				)

				if status := string(event.ResourceStatus); strings.Contains(status, "FAILED") || strings.Contains(status, "DELETE") {
					color.Red(text)
				} else if strings.Contains(status, "UPDATE") {
					color.Yellow(text)
				} else if strings.Contains(status, "CREATE") {
					color.Green(text)
				} else {
					color.Blue(text)
				}

				if iter > 0 && isComplete(stackName, event) {
					return
				}
			}

			token = resp.NextToken
			if token == nil {
				break
			}
		}
	}
}

var completeResourceStatuses = []cloudformation.ResourceStatus{
	cloudformation.ResourceStatusCreateComplete,
	cloudformation.ResourceStatusCreateFailed,
	cloudformation.ResourceStatusUpdateComplete,
	cloudformation.ResourceStatusUpdateFailed,
	cloudformation.ResourceStatusDeleteComplete,
	cloudformation.ResourceStatusDeleteFailed,
}

func isComplete(stackName string, event cloudformation.StackEvent) bool {
	return aws.StringValue(event.LogicalResourceId) == stackName &&
		containsResourceStatus(event, completeResourceStatuses...)
}

func containsResourceStatus(event cloudformation.StackEvent, ss ...cloudformation.ResourceStatus) bool {
	for _, s := range ss {
		if s == event.ResourceStatus {
			return true
		}
	}
	return false
}
