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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Assume the given role
func Assume(config aws.Config, roleARN, sessionName string) (aws.Config, error) {
	api := sts.New(config)
	provider := stscreds.NewAssumeRoleProvider(api, roleARN,
		func(options *stscreds.AssumeRoleProviderOptions) {
			options.RoleSessionName = sessionName
		},
	)

	assumed := config.Copy()
	assumed.Credentials = provider
	return assumed, nil
}
