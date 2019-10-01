/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Subnets: https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeSubnets
type Subnets struct{}

func (Subnets) MarkAndSweep(sess *session.Session, acct string, region string, set *Set) error {
	svc := ec2.New(sess, &aws.Config{Region: aws.String(region)})

	descReq := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("defaultForAz"),
				Values: []*string{aws.String("false")},
			},
		},
	}

	resp, err := svc.DescribeSubnets(descReq)
	if err != nil {
		return err
	}

	for _, sub := range resp.Subnets {
		s := &subnet{Account: acct, Region: region, ID: *sub.SubnetId}
		if set.Mark(s) {
			klog.Warningf("%s: deleting %T: %v", s.ARN(), sub, sub)
			if _, err := svc.DeleteSubnet(&ec2.DeleteSubnetInput{SubnetId: sub.SubnetId}); err != nil {
				klog.Warningf("%v: delete failed: %v", s.ARN(), err)
			}
		}
	}

	return nil
}

func (Subnets) ListAll(sess *session.Session, acct, region string) (*Set, error) {
	svc := ec2.New(sess, &aws.Config{Region: aws.String(region)})
	set := NewSet(0)
	input := &ec2.DescribeSubnetsInput{}

	// Subnets not paginated
	subnets, err := svc.DescribeSubnets(input)
	now := time.Now()
	for _, sn := range subnets.Subnets {
		arn := subnet{
			Account: acct,
			Region:  region,
			ID:      *sn.SubnetId,
		}.ARN()
		set.firstSeen[arn] = now
	}

	return set, errors.Wrapf(err, "couldn't describe subnets for %q in %q", acct, region)
}

type subnet struct {
	Account string
	Region  string
	ID      string
}

func (sub subnet) ARN() string {
	return fmt.Sprintf("arn:aws:ec2:%s:%s:subnet/%s", sub.Region, sub.Account, sub.ID)
}

func (sub subnet) ResourceKey() string {
	return sub.ARN()
}
