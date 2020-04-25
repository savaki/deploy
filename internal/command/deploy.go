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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/savaki/fairy/internal/amazon/role"
	"github.com/savaki/fairy/internal/amazon/stack"
	"github.com/savaki/fairy/internal/banner"
	"github.com/savaki/fairy/internal/command/deploy"
	"github.com/urfave/cli"
)

var deployOptions struct {
	Env      string
	Dir      string
	S3Prefix string
	Project  string
	RoleARN  string
	Version  string
}

var Deploy = cli.Command{
	Name:    "deploy",
	Aliases: []string{"d"},
	Usage:   "deploy resources to cloud provider",
	Action:  deployCommand,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "d,dir",
			Usage:       "dir to resources",
			EnvVar:      "DIR",
			Value:       ".",
			Destination: &deployOptions.Dir,
		},
		cli.StringFlag{
			Name:        "e,env",
			Usage:       "name of environment",
			EnvVar:      "ENV",
			Value:       "local",
			Destination: &deployOptions.Env,
		},
		cli.StringFlag{
			Name:        "prefix",
			Usage:       "prefix for s3 resources",
			EnvVar:      "S3_PREFIX",
			Value:       "resources",
			Destination: &deployOptions.S3Prefix,
		},
		cli.StringFlag{
			Name:        "p,project",
			Usage:       "project name",
			Required:    true,
			EnvVar:      "PROJECT",
			Destination: &deployOptions.Project,
		},
		cli.StringFlag{
			Name:        "r,role",
			Usage:       "role to assume",
			EnvVar:      "ROLE",
			Destination: &deployOptions.RoleARN,
		},
		cli.StringFlag{
			Name:        "version",
			Usage:       "app version",
			EnvVar:      "VERSION",
			Value:       "latest",
			Destination: &deployOptions.Version,
		},
	},
}

func deployCommand(_ *cli.Context) error {
	source, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return fmt.Errorf("unable to load aws config: %w", err)
	}

	target := source
	if deployOptions.RoleARN != "" {
		v, err := role.Assume(source, deployOptions.RoleARN, "fairy")
		if err != nil {
			return fmt.Errorf("unable to assume role, %v: %w", deployOptions.RoleARN, err)
		}
		target = v
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if deployOptions.Project == "" {
		if v := os.Getenv("CODEBUILD_INITIATOR"); v != "" {
			deployOptions.Project = filepath.Base(v)
			fmt.Printf("found CODEBUILD_INITIATOR.  assign project to %v\n", deployOptions.Project)
		}
	}

	banner.Println("deployment fairy started")
	defer func(begin time.Time) {
		banner.Printf("deployment fairy completed - %v\n", time.Now().Sub(begin).Round(time.Millisecond))
	}(time.Now())

	config := deploy.Config{
		Source:  source,
		Target:  target,
		Dir:     deployOptions.Dir,
		Env:     deployOptions.Env,
		Project: deployOptions.Project,
		Parameters: map[string]string{
			stack.Env:      deployOptions.Env,
			stack.S3Prefix: filepath.Join(deployOptions.S3Prefix, deployOptions.Project, deployOptions.Env, deployOptions.Version),
			stack.Version:  deployOptions.Version,
		},
	}

	var fns = []deploy.Func{
		deploy.Bootstrap,
		deploy.Upload,
		deploy.Templates,
	}

	for _, fn := range fns {
		if err := fn(ctx, config); err != nil {
			return err
		}
	}

	return nil
}
