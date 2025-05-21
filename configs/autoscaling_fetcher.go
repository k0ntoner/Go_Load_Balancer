package configs

import (
	"Go_Load_Balancer/models"
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var logger = log.New(os.Stdout, "[AWSClient] ", log.LstdFlags|log.Lmicroseconds)

func fetchInstanceList(groupName string) ([]string, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	autoScalingClient := autoscaling.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	asgResult, err := autoScalingClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{groupName},
	})
	if err != nil {
		return nil, err
	}

	if len(asgResult.AutoScalingGroups) == 0 {
		return nil, nil
	}

	var instanceIDs []string
	for _, inst := range asgResult.AutoScalingGroups[0].Instances {
		instanceIDs = append(instanceIDs, aws.ToString(inst.InstanceId))
	}

	ec2Result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		return nil, err
	}

	var publicIPs []string
	for _, reservation := range ec2Result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PublicIpAddress != nil {
				publicIPs = append(publicIPs, aws.ToString(instance.PublicIpAddress))
			}
		}
	}

	return publicIPs, nil
}

func GetInstances(groupName string) ([]*models.Instance, error) {
	ipAddressList, err := fetchInstanceList(groupName)
	if err != nil {
		return nil, err
	}
	instances := make([]*models.Instance, 0, len(ipAddressList))

	for _, ipAddress := range ipAddressList {
		instances = append(instances, &models.Instance{
			ID:           ipAddress,
			IPAddress:    ipAddress,
			LastUsedTime: time.Time{},
			CountOfLoads: 0,
		})
	}
	return instances, nil
}
