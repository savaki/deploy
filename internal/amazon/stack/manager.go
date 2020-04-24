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
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	gf "github.com/awslabs/goformation/v4"
)

const (
	errResourceNotReady = "ResourceNotReady"
	errValidationError  = "ValidationError"
)

type Manager struct {
	api     cloudformationiface.ClientAPI
	options Options
}

func New(api cloudformationiface.ClientAPI, opts ...Option) *Manager {
	options := buildOptions(opts...)
	return &Manager{
		api:     api,
		options: options,
	}
}

func (m *Manager) Apply(ctx context.Context, changes ...Change) (err error) {
	defer func(begin time.Time) {
		log.Printf("applied %v cloudformation changes (%v) - %v\n",
			len(changes),
			time.Now().Sub(begin).Round(time.Millisecond),
			err,
		)
	}(time.Now())

	// apply all deletes first in reverse order, FILO
	for i := len(changes) - 1; i >= 0; i-- {
		change := changes[i]
		if change.Operation != Delete {
			continue
		}

		if err := m.Delete(ctx, change.Stack.Name); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
	}

	// apply all inserts and updates
	for _, change := range changes {
		switch change.Operation {
		case Insert:
			if err := m.Create(ctx, change.Stack); err != nil {
				return fmt.Errorf("failed to apply changes: %w", err)
			}
		case Update:
			if err := m.Update(ctx, change.Stack); err != nil {
				return fmt.Errorf("failed to apply changes: %w", err)
			}
		}
	}

	return nil
}

func (m *Manager) Create(ctx context.Context, stack Stack) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func(begin time.Time) {
		log.Printf("created cloudformation stack, %v (%v) - %v\n",
			stack.Name,
			time.Now().Sub(begin).Round(time.Millisecond),
			err,
		)
	}(time.Now())
	defer func() {
		if errors.Is(err, context.Canceled) {
			err = nil
		}
	}()

	if m.options.DryRun {
		log.Printf("dry run.  create not applied for stack, %v - %v\n", stack.Name, err)
		return nil
	}

	input := cloudformation.CreateStackInput{
		Capabilities: stack.Capabilities,
		Parameters:   stack.Parameters,
		StackName:    aws.String(stack.Name),
		Tags:         append(m.options.Tags[0:len(m.options.Tags):len(m.options.Tags)], stack.Tags...),
		TemplateBody: aws.String(stack.TemplateBody),
	}
	req := m.api.CreateStackRequest(&input)
	_, err = req.Send(ctx)
	if err != nil {
		return fmt.Errorf("unable to create stack, %v: %w", stack.Name, err)
	}

	go func() {
		defer cancel()
		observeEvents(ctx, m.api, stack.Name)
	}()

	describeInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(stack.Name),
	}
	if err := m.api.WaitUntilStackCreateComplete(ctx, &describeInput); err != nil {
		return fmt.Errorf("failed while waiting for complete to finish for stack, %v: %w", stack.Name, err)
	}

	return nil
}

func (m *Manager) Delete(ctx context.Context, stackName string) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func(begin time.Time) {
		log.Printf("deleted cloudformation stack, %v (%v) - %v\n",
			stackName,
			time.Now().Sub(begin).Round(time.Millisecond),
			err,
		)
	}(time.Now())
	defer func() {
		if errors.Is(err, context.Canceled) {
			err = nil
		}
	}()

	if m.options.DryRun {
		log.Printf("dry run.  delete not applied for stack, %v - %v\n", stackName, err)
		return nil
	}

	req := m.api.DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	if _, err := req.Send(ctx); err != nil {
		return fmt.Errorf("failed to delete stack, %v: %w", stackName, err)
	}

	go func() {
		defer cancel()
		observeEvents(ctx, m.api, stackName)
	}()

	describeInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	if err := m.api.WaitUntilStackUpdateComplete(ctx, &describeInput); err != nil {
		var ae awserr.Error
		if ok := errors.As(err, &ae); ok && ae.Code() == errResourceNotReady {
			return nil
		}
		return fmt.Errorf("failed while waiting for delete to finish for stack, %v: %w", stackName, err)
	}
	return nil
}

func (m *Manager) Exports(ctx context.Context) ([]cloudformation.Export, error) {
	var exports []cloudformation.Export
	var token *string
	for {
		input := cloudformation.ListExportsInput{NextToken: token}
		resp, err := m.api.ListExportsRequest(&input).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to list cloudformation exports: %w", err)
		}
		exports = append(exports, resp.Exports...)

		token = resp.NextToken
		if token == nil {
			break
		}
	}

	return exports, nil
}

func (m *Manager) List(ctx context.Context) (summaries []cloudformation.StackSummary, err error) {
	defer func(begin time.Time) {
		log.Printf("retrieved %v stack summaries, (%v, prefix: %v) - %v\n",
			len(summaries),
			time.Now().Sub(begin).Round(time.Millisecond),
			m.options.Prefix,
			err,
		)
	}(time.Now())

	var token *string
	for {
		req := m.api.ListStacksRequest(&cloudformation.ListStacksInput{NextToken: token})
		resp, err := req.Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		for _, s := range resp.StackSummaries {
			if hasPrefix(*s.StackName, m.options.Prefix) {
				summaries = append(summaries, s)
			}
		}

		token = resp.NextToken
		if token == nil {
			break
		}
	}

	return summaries, nil
}

func (m *Manager) Update(ctx context.Context, stack Stack) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func(begin time.Time) {
		log.Printf("updated cloudformation stack, %v (%v) - %v\n",
			stack.Name,
			time.Now().Sub(begin).Round(time.Millisecond),
			err,
		)
	}(time.Now())
	defer func() {
		if errors.Is(err, context.Canceled) {
			err = nil
		}
	}()

	if m.options.DryRun {
		log.Printf("dry run.  update not applied for stack, %v - %v\n", stack.Name, err)
		return nil
	}

	input := cloudformation.UpdateStackInput{
		Capabilities: stack.Capabilities,
		Parameters:   stack.Parameters,
		StackName:    aws.String(stack.Name),
		Tags:         append(m.options.Tags[0:len(m.options.Tags):len(m.options.Tags)], stack.Tags...),
		TemplateBody: aws.String(stack.TemplateBody),
	}
	req := m.api.UpdateStackRequest(&input)
	_, err = req.Send(ctx)
	if err != nil {
		var ae awserr.Error
		if ok := errors.As(err, &ae); ok && ae.Code() == errValidationError {
			if strings.Contains(ae.Message(), "No updates are to be performed.") {
				log.Printf("skipping update: no updates required\n")
				return nil
			}
		}
		return fmt.Errorf("unable to update stack, %v: %w", stack.Name, err)
	}

	go func() {
		defer cancel()
		observeEvents(ctx, m.api, stack.Name)
	}()

	describeInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(stack.Name),
	}
	if err := m.api.WaitUntilStackUpdateComplete(ctx, &describeInput); err != nil {
		return fmt.Errorf("failed while waiting for update to finish for stack, %v: %w", stack.Name, err)
	}

	return nil
}

func (m *Manager) Upsert(ctx context.Context, stack Stack) error {
	input := cloudformation.GetTemplateInput{
		StackName: aws.String(stack.Name),
	}

	resp, err := m.api.GetTemplateRequest(&input).Send(ctx)
	if err != nil {
		var ae awserr.Error
		if ok := errors.As(err, &ae); ok && ae.Code() == errValidationError {
			if strings.Contains(ae.Message(), "does not exist") {
				return m.Create(ctx, stack)
			}
		}
		return fmt.Errorf("unable to upsert stack, %v: %w", stack.Name, err)
	}

	got, err := gf.ParseYAML([]byte(*resp.TemplateBody))
	if err != nil {
		return fmt.Errorf("unable to upsert stack: unable to parse current template, %v: %w", stack.Name, err)
	}

	want, err := gf.ParseYAML([]byte(stack.TemplateBody))
	if err != nil {
		return fmt.Errorf("unable to upsert stack: unable to parse new template, %v: %w", stack.Name, err)
	}

	if !reflect.DeepEqual(got, want) {
		return m.Update(ctx, stack)
	}

	return nil
}

func hasPrefix(got string, prefixes ...string) bool {
	if len(prefixes) == 0 {
		return true
	}

	for _, p := range prefixes {
		if strings.HasPrefix(got, p) {
			return true
		}
	}

	return false
}
