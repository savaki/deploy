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
	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/stsiface"
	"github.com/savaki/fairy/internal/amazon/role"
	"github.com/savaki/fairy/internal/banner"
	"github.com/savaki/fairy/internal/command/docker"
	"github.com/urfave/cli"
	"log"
	"os"
	"regexp"
	"time"
)

var dockerOptions struct {
	DockerCLI string
	Filename  string
	RoleARN   string
}

var Docker = cli.Command{
	Name:        "docker",
	Usage:       "docker related commands",
	Description: "helpers to simplify interacting with docker and ecr",
	Subcommands: []cli.Command{
		{
			Name:   "login",
			Usage:  "login to ecr docker",
			Action: dockerLoginAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "docker",
					Value:       "docker",
					Usage:       "docker command to invoke",
					EnvVar:      "COMMAND",
					Destination: &dockerOptions.DockerCLI,
				},
				cli.StringFlag{
					Name:        "r,role",
					Usage:       "role to assume",
					EnvVar:      "ROLE",
					Destination: &dockerOptions.RoleARN,
				},
			},
		},
		{
			Name:   "promote",
			Usage:  "promote docker image from source to target account",
			Action: dockerPromoteAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "docker",
					Value:       "docker",
					Usage:       "docker command to invoke",
					EnvVar:      "COMMAND",
					Destination: &dockerOptions.DockerCLI,
				},
				cli.StringFlag{
					Name:        "f,filename",
					Usage:       "properties file containing docker image",
					EnvVar:      "FILENAME",
					Required:    true,
					Destination: &dockerOptions.Filename,
				},
				cli.StringFlag{
					Name:        "r,role",
					Usage:       "role to assume",
					EnvVar:      "ROLE",
					Destination: &dockerOptions.RoleARN,
				},
			},
		},
	},
}

func dockerLoginAction(_ *cli.Context) error {
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

	return docker.Login(ctx, ecr.New(target), dockerOptions.DockerCLI)
}

func dockerPromoteAction(_ *cli.Context) (err error) {
	if _, err := os.Stat(dockerOptions.Filename); os.IsNotExist(err) {
		return nil
	}

	banner.Println("docker promote started")
	defer func(begin time.Time) {
		banner.Printf("docker promote completed (%v) - %v\n", time.Now().Sub(begin).Round(time.Millisecond), err)
	}(time.Now())

	var content struct {
		Image string `toml:"IMAGE"`
	}
	if _, err := toml.DecodeFile(dockerOptions.Filename, &content); err != nil {
		return fmt.Errorf("unable to promote docker image: failed to decode file, %v: %w", dockerOptions.Filename, err)
	}

	image, err := parseImage(content.Image)
	if err != nil {
		return fmt.Errorf("unable to promote docker image: unable to parse IMAGE, %v, in file, %v: %w", content.Image, dockerOptions.Filename, err)
	}
	if content.Image == "" {
		return fmt.Errorf("unable to promote docker image: invalid IMAGE, %v, in file, %v", content.Image, dockerOptions.Filename)
	}

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

	sourceAPI := ecr.New(source)
	targetAPI := ecr.New(target)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ok, err := hasVersion(ctx, targetAPI, image.RepositoryName, image.Tag)
	if err != nil {
		return fmt.Errorf("unable to promote image: %w", err)
	}
	if ok {
		log.Println("no need to promote image.  image already exists in target account.")
		return nil
	}

	if err := docker.Login(ctx, sourceAPI, dockerOptions.DockerCLI); err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	if err := docker.Pull(dockerOptions.DockerCLI, content.Image); err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	accountID, err := getAccountID(ctx, sts.New(target))
	if err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	if err := createRepositoryIfNotExists(ctx, targetAPI, image.RepositoryName); err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	image.AccountID = accountID
	targetImage := image.String()
	if err := docker.Tag(dockerOptions.DockerCLI, content.Image, targetImage); err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	if err := docker.Push(dockerOptions.DockerCLI, targetImage); err != nil {
		return fmt.Errorf("unable to promote image, %v: %w", content.Image, err)
	}

	return nil
}

func createRepositoryIfNotExists(ctx context.Context, api ecriface.ClientAPI, repositoryName string) error {
	repos, err := listRepositories(ctx, api, repositoryName)
	if err != nil {
		return fmt.Errorf("unable to create repository, %v: %w", repositoryName, err)
	}

	if len(repos) > 0 {
		return nil
	}

	input := ecr.CreateRepositoryInput{
		ImageTagMutability: ecr.ImageTagMutabilityImmutable,
		RepositoryName:     aws.String(repositoryName),
	}
	if _, err := api.CreateRepositoryRequest(&input).Send(ctx); err != nil {
		return fmt.Errorf("failed to create repository, %v: %w", repositoryName, err)
	}

	log.Printf("successfully created repository, %v\n", repositoryName)

	return nil
}

func getAccountID(ctx context.Context, api stsiface.ClientAPI) (string, error) {
	input := sts.GetCallerIdentityInput{}
	resp, err := api.GetCallerIdentityRequest(&input).Send(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get account id, %w", err)
	}

	return aws.StringValue(resp.Account), nil
}

func hasVersion(ctx context.Context, api ecriface.ClientAPI, repositoryName, tag string) (bool, error) {
	identifiers, err := listImages(ctx, api, repositoryName)
	if err != nil {
		return false, fmt.Errorf("has version failed: %w", err)
	}

	for _, i := range identifiers {
		if aws.StringValue(i.ImageTag) == tag {
			return true, nil
		}
	}
	return false, nil
}

func listImages(ctx context.Context, api ecriface.ClientAPI, repositoryName string) ([]ecr.ImageIdentifier, error) {
	var identifiers []ecr.ImageIdentifier
	var token *string
	for {
		input := ecr.ListImagesInput{
			Filter:         &ecr.ListImagesFilter{TagStatus: ecr.TagStatusTagged},
			NextToken:      token,
			RepositoryName: aws.String(repositoryName),
		}
		resp, err := api.ListImagesRequest(&input).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list images from repository, %v: %w", repositoryName, err)
		}

		identifiers = append(identifiers, resp.ImageIds...)

		token = resp.NextToken
		if token == nil {
			break
		}
	}

	return identifiers, nil
}

func listRepositories(ctx context.Context, api ecriface.ClientAPI, repositoryNames ...string) ([]ecr.Repository, error) {
	var repos []ecr.Repository
	var token *string
	for {
		input := ecr.DescribeRepositoriesInput{
			NextToken:       token,
			RepositoryNames: repositoryNames,
		}
		resp, err := api.DescribeRepositoriesRequest(&input).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}
		repos = append(repos, resp.Repositories...)

		token = resp.NextToken
		if token == nil {
			break
		}
	}
	return repos, nil
}

type Image struct {
	AccountID      string
	Region         string
	RepositoryName string
	Tag            string
}

func (i Image) String() string {
	return fmt.Sprintf("%v.dkr.ecr.%v.amazonaws.com/%v:%v", i.AccountID, i.Region, i.RepositoryName, i.Tag)
}

var reImage = regexp.MustCompile(`^(\d+).dkr.ecr.([^.]+).amazonaws.com/([^:]+):(.*)$`)

func parseImage(s string) (Image, error) {
	match := reImage.FindStringSubmatch(s)
	if len(match) != 5 {
		return Image{}, fmt.Errorf("invalid docker image, %v", s)
	}

	return Image{
		AccountID:      match[1],
		Region:         match[2],
		RepositoryName: match[3],
		Tag:            match[4],
	}, nil
}
