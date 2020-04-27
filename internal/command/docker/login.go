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

package docker

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
)

func Login(ctx context.Context, api ecriface.ClientAPI, dockerCLI string) error {
	log.Printf("## %v login\n", dockerCLI)

	input := ecr.GetAuthorizationTokenInput{}
	resp, err := api.GetAuthorizationTokenRequest(&input).Send(ctx)
	if err != nil {
		return fmt.Errorf("failed to login to docker: %w", err)
	}

	for _, token := range resp.AuthorizationData {
		data, err := base64.StdEncoding.DecodeString(aws.StringValue(token.AuthorizationToken))
		if err != nil {
			return fmt.Errorf("failed to decode authorization token: %w", err)
		}

		password := string(data)
		if prefix := "AWS:"; strings.HasPrefix(password, prefix) {
			password = password[len(prefix):]
		}

		cmd := exec.Command(dockerCLI, "login", "-u", "AWS", "--password-stdin", aws.StringValue(token.ProxyEndpoint))
		cmd.Stdin = strings.NewReader(password)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}
	}

	return nil
}
