package deploy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/servicediscoveryiface"
	"github.com/savaki/fairy/internal/amazon/stack"
	"github.com/savaki/fairy/internal/banner"
)

func CloudMapNamespaceIfNotExists(ctx context.Context, config Config) error {
	banner.Println("creating cloudmap namespace ...")

	api := servicediscovery.New(config.Target)

	var ae awserr.Error
	if nss, err := listNamespaces(ctx, api, config.Env); err == nil && len(nss) > 0 {
		ns := nss[0]
		log.Printf("using existing cloudmap namespace, %v (%v)\n", aws.StringValue(ns.Name), aws.StringValue(ns.Id))
		config.Parameters[stack.CloudMapNamespaceARN] = aws.StringValue(ns.Arn)
		return nil
	} else if ok := errors.As(err, &ae); !ok || ae.Code() != servicediscovery.ErrCodeNamespaceNotFound {
		return fmt.Errorf("failed to request cloudmap namespace, %v: %w", config.Env, err)
	}

	if config.VpcID == "" {
		log.Println("vpc not set.  cloudmap namespace will not be created.")
		return nil
	}

	input := servicediscovery.CreatePrivateDnsNamespaceInput{
		CreatorRequestId: aws.String(strconv.FormatInt(time.Now().UnixNano(), 36)),
		Description:      aws.String(fmt.Sprintf("service discovery for %v env", config.Env)),
		Name:             aws.String(config.Env),
		Vpc:              aws.String(config.VpcID),
	}
	resp, err := api.CreatePrivateDnsNamespaceRequest(&input).Send(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cloudmap namespace, %v: %w", config.Env, err)
	}

	for {
		log.Println("waiting for cloudmap namespace to be created ...")
		select {
		case <-time.After(6 * time.Second):
			// ok
		case <-ctx.Done():
			return ctx.Err()
		}

		input := servicediscovery.GetOperationInput{OperationId: resp.OperationId}
		op, err := api.GetOperationRequest(&input).Send(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve operation, %v: %w", aws.StringValue(resp.OperationId), err)
		}

		switch op.Operation.Status {
		case servicediscovery.OperationStatusSuccess:
			log.Println("created cloudmap namespace,", config.Env)
			return nil
		case servicediscovery.OperationStatusFail:
			return fmt.Errorf("unable to create cloudmap namespace, %v: %w",
				config.Env,
				awserr.New(aws.StringValue(op.Operation.ErrorCode), aws.StringValue(op.Operation.ErrorMessage), nil),
			)
		}
	}
}

func listNamespaces(ctx context.Context, api servicediscoveryiface.ClientAPI, envs ...string) ([]servicediscovery.NamespaceSummary, error) {
	var summaries []servicediscovery.NamespaceSummary
	var token *string
	for {
		input := servicediscovery.ListNamespacesInput{NextToken: token}
		resp, err := api.ListNamespacesRequest(&input).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list cloudmap namespaces: %w", err)
		}

		for _, ns := range resp.Namespaces {
			if len(envs) == 0 || containsString(envs, aws.StringValue(ns.Name)) {
				summaries = append(summaries, ns)
			}
		}

		token = resp.NextToken
		if token == nil {
			break
		}
	}
	return summaries, nil
}

func containsString(ss []string, value string) bool {
	for _, s := range ss {
		if s == value {
			return true
		}
	}
	return false
}
