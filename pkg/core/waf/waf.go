package waf

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	wafLib "github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

func UpdateIPSet(ctx context.Context, cfg aws.Config, ipSetName string, cidrs string) error {
	wafHandler := wafLib.NewFromConfig(cfg)

	ipSetsList, err := wafHandler.ListIPSets(ctx, &wafLib.ListIPSetsInput{
		Scope: types.ScopeCloudfront,
	})

	if err != nil {
		return err
	}

	for _, ipSet := range ipSetsList.IPSets {
		if *ipSet.Name == ipSetName {
			ipSetDetails, err := wafHandler.GetIPSet(ctx, &wafLib.GetIPSetInput{
				Id:    ipSet.Id,
				Name:  ipSet.Name,
				Scope: types.ScopeCloudfront,
			})
			if err != nil {
				return err
			}

			_, err = wafHandler.UpdateIPSet(ctx, &wafLib.UpdateIPSetInput{
				Addresses: append(ipSetDetails.IPSet.Addresses, strings.Split(cidrs, ",")...),
				Id:        ipSet.Id,
				LockToken: ipSet.LockToken,
				Name:      ipSet.Name,
				Scope:     types.ScopeCloudfront,
			})

			if err != nil {
				return err
			}

			return nil
		}
	}

	return errors.New("nothing to do")
}
