/*
EC2 Instances Control Wrapper

2019 © Postgres.ai
Based on source of docker-machine cli tool

https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/
*/

package ec2ctrl

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"../log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/docker/machine/libmachine/ssh"
)

const (
	DefaultSshUser                  = "ubuntu"
	DefaultIpRange                  = "0.0.0.0/0"
	DefaultSshPort                  = 22
	DefaultSecurityGroup            = "ec2ctrl-docker-machine"
	DefaultSecurityGroupDescription = "EC2Ctrl Docker Machine"
	DefaultDockerPort               = 2376
	DefaultSwarmPort                = 3376
)

type Region struct {
	AmiId string
}

var RegionDetails map[string]*Region = map[string]*Region{
	"ap-northeast-1":  {"ami-bcb7f6da"},
	"ap-northeast-2":  {"ami-5073de3e"},
	"ap-southeast-1":  {"ami-41e4af3d"},
	"ap-southeast-2":  {"ami-c1498fa3"},
	"ap-south-1":      {"ami-1083dc7f"},
	"ca-central-1":    {"ami-8d9e19e9"},
	"cn-north-1":      {"ami-cc4499a1"}, // Note: this is 20180126
	"cn-northwest-1":  {"ami-fd0e1a9f"}, // Note: this is 20180126
	"eu-north-1":      {"ami-017ff17f"},
	"eu-central-1":    {"ami-bc4925d3"},
	"eu-west-1":       {"ami-0b541372"},
	"eu-west-2":       {"ami-ff46a298"},
	"eu-west-3":       {"ami-9465d3e9"},
	"sa-east-1":       {"ami-b5501bd9"},
	"us-east-1":       {"ami-927185ef"},
	"us-east-2":       {"ami-b9daeddc"},
	"us-west-1":       {"ami-264c4646"},
	"us-west-2":       {"ami-78a22900"},
	"us-gov-west-1":   {"ami-2561ea44"},
	"custom-endpoint": {""},
}

type Ec2Configuration struct {
	AwsInstanceType         string `yaml:"awsInstanceType"`
	AwsRegion               string `yaml:"awsRegion"`
	AwsZone                 string `yaml:"awsZone"`
	AwsBlockDurationMinutes int64  `yaml:"awsBlockDurationMinutes"`
	AwsKeyName              string `yaml:"awsKeyName"`
	AwsKeyPath              string `yaml:"awsKeyPath"`
}

type Ec2Ctrl struct {
	configuration           Ec2Configuration
	ec2Client               *ec2.EC2
	sshClient               ssh.Client
	securityGroupName       string
	vpcId                   string
	subNetId                string
	securityGroupIds        []string
	requestId               string
	instanceId              string
	publicInstanceIpAddress string
}

func NewEc2Ctrl(conf Ec2Configuration) *Ec2Ctrl {
	ec2Client := getClient(conf)
	return &Ec2Ctrl{
		configuration:           conf,
		ec2Client:               ec2Client,
		securityGroupName:       DefaultSecurityGroup,
		vpcId:                   "",
		subNetId:                "",
		instanceId:              "",
		requestId:               "",
		publicInstanceIpAddress: "",
	}
}

func getClient(conf Ec2Configuration) *ec2.EC2 {
	awsConfig := aws.NewConfig()
	awsConfig = awsConfig.WithRegion(conf.AwsRegion)
	ec2Client := ec2.New(session.New(awsConfig))
	return ec2Client
}

func (e *Ec2Ctrl) GetPublicInstanceIpAddress(instanceId string) (string, error) {
	if e.instanceId == instanceId && e.publicInstanceIpAddress != "" {
		return e.publicInstanceIpAddress, nil
	}
	return "", fmt.Errorf("Unable to get a instance ip. May be instance not started.")
}

func (e *Ec2Ctrl) GetEc2Client() *ec2.EC2 {
	if e.ec2Client == nil {
		awsConfig := aws.NewConfig()
		e.ec2Client = ec2.New(session.New(awsConfig))
	}
	return e.ec2Client
}

// Get the minimum price of ec2 spot instance with the specified type.
// The minimum value of price for 24 hours.
func (e *Ec2Ctrl) GetHistoryInstancePrice() float64 {
	duration, _ := time.ParseDuration("-24h") // 24*30 hours
	endTime := time.Now()
	startTime := endTime.Add(duration)

	input := &ec2.DescribeSpotPriceHistoryInput{
		EndTime: &endTime,
		InstanceTypes: []*string{
			aws.String(e.configuration.AwsInstanceType),
		},
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: &startTime,
	}

	results, err := e.GetEc2Client().DescribeSpotPriceHistory(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Err(aerr.Error())
			}
		} else {
			log.Err(err.Error())
		}
		return 0.0
	}

	minPrice := 100.0
	maxPrice := 0.0
	for _, item := range results.SpotPriceHistory {
		itemPrice, err := strconv.ParseFloat(*item.SpotPrice, 64)
		if err == nil {
			if itemPrice < minPrice {
				minPrice = itemPrice
			}
			if maxPrice < itemPrice {
				maxPrice = itemPrice
			}
		}
	}

	return minPrice
}

// Create ec2 spot instance of the specified type with 16GiB GP2 drive
func (e *Ec2Ctrl) CreateSpotInstanceRequest(price float64) (string, error) {
	bdm := &ec2.BlockDeviceMapping{
		DeviceName: aws.String("/dev/sda1"),
		Ebs: &ec2.EbsBlockDevice{
			VolumeSize:          aws.Int64(16),
			VolumeType:          aws.String("gp2"),
			DeleteOnTermination: aws.Bool(true),
		},
	}
	subNetId, err := e.GetSubNet()
	if err != nil {
		return "", fmt.Errorf("Unable to find a subnet.")
	}
	e.subNetId = subNetId
	e.configureSecurityGroups([]string{e.securityGroupName})
	netSpecs := []*ec2.InstanceNetworkInterfaceSpecification{{
		DeviceIndex:              aws.Int64(0), // eth0
		Groups:                   makePointerSlice(e.securityGroupIds),
		SubnetId:                 &e.subNetId,
		AssociatePublicIpAddress: aws.Bool(true),
	}}
	image := RegionDetails[e.configuration.AwsRegion].AmiId
	input := &ec2.RequestSpotInstancesInput{
		BlockDurationMinutes: &e.configuration.AwsBlockDurationMinutes,
		LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
			ImageId: aws.String(image),
			Placement: &ec2.SpotPlacement{
				AvailabilityZone: aws.String(e.configuration.AwsRegion + e.configuration.AwsZone),
			},
			KeyName:           aws.String(e.configuration.AwsKeyName),
			InstanceType:      aws.String(e.configuration.AwsInstanceType),
			NetworkInterfaces: netSpecs,
			Monitoring:        &ec2.RunInstancesMonitoringEnabled{Enabled: aws.Bool(true)},
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Arn: aws.String(""),
			},
			EbsOptimized:        aws.Bool(true),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{bdm},
		},
		InstanceCount: aws.Int64(1),
		SpotPrice:     aws.String(strconv.FormatFloat(price, 'f', 6, 64)),
	}

	spotInstanceRequest, err := e.GetEc2Client().RequestSpotInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Err(aerr.Error())
			}
		} else {
			log.Err(err.Error())
		}
		return "", err
	}
	e.requestId = *spotInstanceRequest.SpotInstanceRequests[0].SpotInstanceRequestId
	return e.requestId, nil
}

func (e *Ec2Ctrl) CreateSpotInstance(price float64) (string, error) {
	_, ierr := e.CreateSpotInstanceRequest(price)
	if ierr != nil {
		return "", fmt.Errorf("Unable to create spot instances request.")
	}
	// wait for request ready
	if err := mcnutils.WaitFor(e.spotInstanceRequestReady()); err != nil {
		e.CancelSpotInstanceReuest(e.requestId)
		return "", err
	}
	// wait for instance ready
	if err := mcnutils.WaitFor(e.spotInstanceReady()); err != nil {
		e.CancelSpotInstanceReuest(e.requestId)
		e.TerminateInstance(e.instanceId)
		return "", err
	}
	return e.instanceId, nil
}

// Check of spot request ready
func (e *Ec2Ctrl) spotInstanceRequestReady() func() bool {
	return func() bool {
		requestDesription, rdErr := e.GetSpotInstanceSpotRequestDescription(e.requestId)
		if rdErr != nil {
			return false
		}
		var result bool
		switch *requestDesription.SpotInstanceRequests[0].Status.Code {
		case "price-too-low":
			log.Dbg("Instance price too low. Detecting actual price...")
			actualPrice, _ := e.GetActualInstancePriceFromMessage(*requestDesription.SpotInstanceRequests[0].Status.Message)
			log.Dbg("Actual instance price is ", actualPrice)
			log.Dbg("Canceling unsuccessful spot request...")
			e.CancelSpotInstanceReuest(e.requestId)
			log.Dbg("Creating new spot request with actual price...")
			requestId, err := e.CreateSpotInstanceRequest(actualPrice)
			if err != nil {
				log.Err("Error happened during creating instance with actual price", err)
				return false
			}
			e.requestId = requestId
			log.Dbg("Waiting for instance ready...")
			result = false
		case "request-canceled-and-instance-running":
			result = true
		case "fulfilled":
			e.instanceId = *requestDesription.SpotInstanceRequests[0].InstanceId
			result = true
		}
		return result
	}
}

// Check of spot instance ready
func (e *Ec2Ctrl) spotInstanceReady() func() bool {
	return func() bool {
		instance, ierr := e.GetInstanceDescription(e.instanceId)
		if ierr == nil {
			e.instanceId = *instance.InstanceId
			if instance.PublicIpAddress != nil {
				e.publicInstanceIpAddress = *(instance.PublicIpAddress)
			}
			instanceState := instance.State.Name
			if *instanceState == "running" {
				return true
			}
		}
		return false
	}
}

// Get ec2 spot instance description дspecified request id
func (e *Ec2Ctrl) GetSpotInstanceSpotRequestDescription(spotInstanceRequestId string) (*ec2.DescribeSpotInstanceRequestsOutput, error) {
	spotInstanceRequestDescritpion, err := e.GetEc2Client().DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{&spotInstanceRequestId},
	})
	if err != nil {
		// Unexpected; no need to retry
		return nil, fmt.Errorf("Error describing previously made spot instance request: %v.", err)
	}
	return spotInstanceRequestDescritpion, nil
}

// Cancel ec2 spot instance request by specified request id
func (e *Ec2Ctrl) CancelSpotInstanceReuest(spotInstanceRequestId string) (bool, error) {
	if spotInstanceRequestId == "" {
		return false, fmt.Errorf("Empty value of spotInstanceRequestId.")
	}

	input := &ec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{
			aws.String(spotInstanceRequestId),
		},
	}

	_, err := e.GetEc2Client().CancelSpotInstanceRequests(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Err(aerr.Error())
			}
		} else {
			log.Err(err.Error())
		}
		return false, err
	}

	return true, nil
}

// Get actual ec2 spot instance price from error message
func (e *Ec2Ctrl) GetActualInstancePriceFromMessage(message string) (float64, error) {
	actualPriceRegExp := regexp.MustCompile(`[0-9]+[.][0-9]+`)
	prices := actualPriceRegExp.FindAllString(message, -1)
	actualPriceStr := prices[len(prices)-1]
	actualPrice, err := strconv.ParseFloat(actualPriceStr, 64)
	return actualPrice, err
}

// Get ec2 spot instance description by specified instance id
func (e *Ec2Ctrl) GetInstanceDescription(instanceId string) (*ec2.Instance, error) {
	instances, err := e.GetEc2Client().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{&instanceId},
	})
	if err != nil {
		return nil, err
	}
	if len(instances.Reservations) > 0 {
		instance := instances.Reservations[0].Instances[0]
		return instance, nil
	} else {
		return nil, fmt.Errorf("Instance not found.")
	}
}

// Get deafult ec2 vpc id
func (e *Ec2Ctrl) GetDefaultVpcId() (string, error) {
	output, err := e.GetEc2Client().DescribeAccountAttributes(&ec2.DescribeAccountAttributesInput{})
	if err != nil {
		return "", err
	}

	for _, attribute := range output.AccountAttributes {
		if *attribute.AttributeName == "default-vpc" {
			value := *attribute.AttributeValues[0].AttributeValue
			if value == "none" {
				return "", errors.New("default-vpc is 'none'")
			}
			return value, nil
		}
	}

	return "", errors.New("No default-vpc attribute")
}

// Get sub net
func (e *Ec2Ctrl) GetSubNet() (string, error) {
	vpcId, err := e.GetDefaultVpcId()
	if err != nil {
		return "", errors.New("default-vpc is not found")
	}
	e.vpcId = vpcId
	regionZone := e.configuration.AwsRegion + e.configuration.AwsZone
	filters := []*ec2.Filter{
		{
			Name:   aws.String("availability-zone"),
			Values: []*string{&regionZone},
		},
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{&vpcId},
		},
	}

	subnets, err := e.GetEc2Client().DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return "", err
	}

	if len(subnets.Subnets) == 0 {
		return "", fmt.Errorf("Unable to find a subnet that is both in the zone %s"+
			" and belonging to VPC ID %s.", regionZone, vpcId)
	}

	subNetId := *subnets.Subnets[0].SubnetId

	// try to find default
	if len(subnets.Subnets) > 1 {
		for _, subnet := range subnets.Subnets {
			if subnet.DefaultForAz != nil && *subnet.DefaultForAz {
				subNetId = *subnet.SubnetId
				break
			}
		}
	}

	return subNetId, nil
}

// Make array of string pointers from array of strings
func makePointerSlice(stackSlice []string) []*string {
	pointerSlice := []*string{}
	for i := range stackSlice {
		pointerSlice = append(pointerSlice, &stackSlice[i])
	}
	return pointerSlice
}

// Check of existance security group
func (e *Ec2Ctrl) securityGroupAvailableFunc(id string) func() bool {
	return func() bool {

		securityGroup, err := e.GetEc2Client().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{&id},
		})
		if err == nil && len(securityGroup.SecurityGroups) > 0 {
			return true
		} else if err == nil {
			return false
		}
		log.Dbg(err)
		return false
	}
}

// Configure exisitng or create new security group and configure it
func (e *Ec2Ctrl) configureSecurityGroups(groupNames []string) error {
	if len(groupNames) == 0 {
		return nil
	}

	filters := []*ec2.Filter{
		{
			Name:   aws.String("group-name"),
			Values: makePointerSlice(groupNames),
		},
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{&e.vpcId},
		},
	}
	groups, err := e.GetEc2Client().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return err
	}

	var groupsByName = make(map[string]*ec2.SecurityGroup)
	for _, securityGroup := range groups.SecurityGroups {
		groupsByName[*securityGroup.GroupName] = securityGroup
	}

	for _, groupName := range groupNames {
		var group *ec2.SecurityGroup
		securityGroup, ok := groupsByName[groupName]
		if ok {
			group = securityGroup
		} else {
			groupResp, err := e.GetEc2Client().CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
				GroupName:   aws.String(groupName),
				Description: aws.String("Docker Machine"),
				VpcId:       aws.String(e.vpcId),
			})
			if err != nil {
				return err
			}
			// Manually translate into the security group construct
			group = &ec2.SecurityGroup{
				GroupId:   groupResp.GroupId,
				VpcId:     aws.String(e.vpcId),
				GroupName: aws.String(groupName),
			}
			// wait until created (dat eventual consistency)
			//log.Debugf("waiting for group (%s) to become available", *group.GroupId)
			if err := mcnutils.WaitFor(e.securityGroupAvailableFunc(*group.GroupId)); err != nil {
				return err
			}
		}
		e.securityGroupIds = append(e.securityGroupIds, *group.GroupId)

		perms, err := e.configureSecurityGroupPermissions(group)
		if err != nil {
			return err
		}

		if len(perms) != 0 {
			log.Dbg("Authorizing group %s with permissions: ", groupNames, perms)
			_, err := e.GetEc2Client().AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:       group.GroupId,
				IpPermissions: perms,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Configure security group permisions
func (e *Ec2Ctrl) configureSecurityGroupPermissions(group *ec2.SecurityGroup) ([]*ec2.IpPermission, error) {
	hasPorts := make(map[string]bool)
	for _, p := range group.IpPermissions {
		if p.FromPort != nil {
			hasPorts[fmt.Sprintf("%d/%s", *p.FromPort, *p.IpProtocol)] = true
		}
	}

	perms := []*ec2.IpPermission{}

	if !hasPorts[fmt.Sprintf("%d/tcp", DefaultSshPort)] {
		perms = append(perms, &ec2.IpPermission{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(int64(DefaultSshPort)),
			ToPort:     aws.Int64(int64(DefaultSshPort)),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String(DefaultIpRange)}},
		})
	}

	if !hasPorts[fmt.Sprintf("%d/tcp", DefaultDockerPort)] {
		perms = append(perms, &ec2.IpPermission{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(int64(DefaultDockerPort)),
			ToPort:     aws.Int64(int64(DefaultDockerPort)),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String(DefaultIpRange)}},
		})
	}

	return perms, nil
}

// Terminate Ec2 instance by specified instance id
func (e *Ec2Ctrl) TerminateInstance(instanceId string) (bool, error) {
	if instanceId == "" {
		return false, fmt.Errorf("Empty value of instanceId.")
	}

	_, err := e.GetEc2Client().TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instanceId},
	})

	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown instance") ||
			strings.HasPrefix(err.Error(), "InvalidInstanceID.NotFound") {
			return false, fmt.Errorf("Remote instance does not exist, proceeding with removing local reference.")
		}

		return false, fmt.Errorf("Unable to terminate instance: %s.", err)
	}
	log.Dbg("Instance " + e.instanceId + " terminated")
	if e.instanceId == instanceId {
		e.instanceId = ""
		if e.requestId != "" {
			rres, rerr := e.CancelSpotInstanceReuest(e.requestId)
			if rres && rerr == nil {
				log.Dbg("Instance request " + e.requestId + " canceled")
			} else {
				return false, fmt.Errorf("Instance terminated but can't cancel request. %v", err)
			}
			e.requestId = ""
		}
		return true, nil
	}

	return true, nil
}

// Check that instance is running and ready by specified instance id
func (e *Ec2Ctrl) IsInstanceRunning(instanceId string) (bool, error) {
	instance, ferr := e.GetInstanceDescription(instanceId)
	if ferr != nil {
		return false, fmt.Errorf("Unable to check instance state %v.", ferr)
	}
	if instance != nil && instance.State != nil && instance.State.Name != nil {
		if *instance.State.Name == "running" {
			if e.instanceId == "" {
				e.instanceId = *(instance.InstanceId)
				e.publicInstanceIpAddress = *(instance.PublicIpAddress)
			}
		}
		return *instance.State.Name == "running", nil
	}
	return false, nil
}

// Get SSH client for instance
func (e *Ec2Ctrl) GetInstanceSshClient(instanceId string) (ssh.Client, error) {
	if _, err := os.Stat(e.configuration.AwsKeyPath); err != nil {
		return nil, fmt.Errorf("File given as AwsKeyPath does not exist.")
	}
	if e.instanceId != instanceId {
		return nil, fmt.Errorf("Unable to get a instance SSH client. May be instance not started.")
	}
	address, err := e.GetPublicInstanceIpAddress(instanceId)
	if err != nil {
		return nil, err
	}
	port := DefaultSshPort
	var auth *ssh.Auth
	if e.configuration.AwsKeyPath == "" {
		auth = &ssh.Auth{}
	} else {
		auth = &ssh.Auth{
			Keys: []string{e.configuration.AwsKeyPath},
		}
	}
	client, err := ssh.NewClient(DefaultSshUser, address, port, auth)
	e.sshClient = client
	return client, err
}

// Execute command on instance via SSH
func (e *Ec2Ctrl) RunInstanceSshCommand(command string, showStatus bool) (string, error) {
	client := e.sshClient
	if client == nil {
		return "", fmt.Errorf("Instance SSH client not available yet.")
	}
	log.Dbg("EC2 SSH command: " + command)
	output, err := client.Output(command)
	if err != nil {
		if showStatus {
			log.Err(command + " " + log.FAIL)
		}
		return "", fmt.Errorf(`ssh command error:\ncommand : %s\nerr     : %v\noutput  : %s`, command, err, output)
	}
	if showStatus {
		log.Msg(command + " " + log.OK)
	}

	return output, nil
}

func (e *Ec2Ctrl) sshAvailableFunc() func() bool {
	return func() bool {
		log.Dbg("Waiting for SSH access...")
		if _, err := e.RunInstanceSshCommand("exit 0", false); err != nil {
			log.Err("Error getting ssh command 'exit 0' : %s", err)
			return false
		}
		return true
	}
}

// Wait SSH access to instance
func (e *Ec2Ctrl) WaitInstanceForSsh() error {
	if e.instanceId == "" || e.sshClient == nil {
		return fmt.Errorf("SSH client is not initialized.")
	}
	// Try to dial SSH for 30 seconds before timing out.
	if err := mcnutils.WaitFor(e.sshAvailableFunc()); err != nil {
		return fmt.Errorf("Too many retries waiting for SSH to be available.  Last error: %s", err)
	}
	return nil
}

// Attach EC2 volume to EC2 instance by specified isntance and volume ids
func (e *Ec2Ctrl) AttachInstanceVolume(instanceId string, ebsVolumeId string, mountPoint string) (bool, error) {
	input := &ec2.AttachVolumeInput{
		Device:     aws.String(mountPoint),
		InstanceId: aws.String(instanceId),
		VolumeId:   aws.String(ebsVolumeId),
	}

	result, err := e.GetEc2Client().AttachVolume(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Err(aerr.Error())
				return false, aerr
			}
		} else {
			log.Err(err.Error())
			return false, err
		}
	}
	log.Dbg("AttachVolume result:", result)

	return true, nil
}
