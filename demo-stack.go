package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

func SecurityGroupStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	vpcId := "vpc-535bd136"
	vpcName := "lookupvpc"
	vpc := awsec2.Vpc_FromLookup(stack, &vpcName, &awsec2.VpcLookupOptions{VpcId: &vpcId})
	asgSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("AsgSecurityGroup"), &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: jsii.String("AsgSecurityGroup"),
		Description:       jsii.String("Sg for auto scaling group"),
	})

	loadBalancerSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("LoadbalancerSecurityGroup"), &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: jsii.String("LbSecurityGroup"),
		Description:       jsii.String("Sg for lb group"),
	})

	loadBalancerSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(443)),
		jsii.String("Default HTTPS Port"),
		jsii.Bool(false),
	)

	loadBalancerSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Default HTTP Port"),
		jsii.Bool(false),
	)

	asgSecurityGroup.AddIngressRule(awsec2.Peer_SecurityGroupId(jsii.String(
		*loadBalancerSecurityGroup.SecurityGroupId()),
		jsii.String(*awscdk.Aws_ACCOUNT_ID())),
		awsec2.Port_AllTraffic(),
		jsii.String("Enable All Port from loadbalancer"),
		jsii.Bool(false),
	)
	return stack
}

func TagretGroupStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	vpcId := "vpc-535bd136"
	vpcName := "lookupvpc"
	vpc := awsec2.Vpc_FromLookup(stack, &vpcName, &awsec2.VpcLookupOptions{VpcId: &vpcId})

	awselasticloadbalancingv2.ApplicationLoadBalancer_FromLookup(stack, jsii.String("LookupConstruct"),
		&awselasticloadbalancingv2.ApplicationLoadBalancerLookupOptions{LoadBalancerArn: jsii.String("arn:aws:elasticloadbalancing:us-east-1:305251478828:loadbalancer/app/Outlinewikilb/d7d9340dc5f87faf")})

	awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String("TargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{TargetType: awselasticloadbalancingv2.TargetType_IP,
		TargetGroupName: jsii.String("DemoTg"), Vpc: vpc, Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{Enabled: jsii.Bool(true), HealthyHttpCodes: jsii.String("200"), Path: jsii.String("/"), Interval: awscdk.Duration_Seconds(jsii.Number(30))}})
	return stack
}

func CdkConsrtuctStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	regionName := *awscdk.Aws_REGION()
	var sprops awscdk.StackProps
	vpcId := "vpc-535bd136"
	sgId := "sg-f406fe90"
	sgName := "Default"
	vpcName := "lookupvpc"
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := awsec2.Vpc_FromLookup(stack, &vpcName, &awsec2.VpcLookupOptions{VpcId: &vpcId})

	image := awsec2.NewAmazonLinuxImage(&awsec2.AmazonLinuxImageProps{CpuType: awsec2.AmazonLinuxCpuType_X86_64, Edition: awsec2.AmazonLinuxEdition_STANDARD,
		Generation: awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2, Virtualization: awsec2.AmazonLinuxVirt_HVM, Kernel: awsec2.AmazonLinuxKernel_KERNEL5_X})

	securityGroup := awsec2.SecurityGroup_FromLookupById(stack, jsii.String(sgName), jsii.String(sgId))

	ec2Instance := awsec2.NewInstance(stack, jsii.String("InstanceBrz"), &awsec2.InstanceProps{
		Vpc: vpc, PropagateTagsToVolumeOnCreation: jsii.Bool(true),
		InstanceType:  awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE2, awsec2.InstanceSize_MICRO),
		MachineImage:  image,
		KeyName:       jsii.String("breezethru-demo-key-pair"),
		SecurityGroup: securityGroup,
		InstanceName:  jsii.String("brz-demo-" + regionName),
		UserData:      awsec2.UserData_ForLinux(&awsec2.LinuxUserDataOptions{Shebang: jsii.String("#!/bin/bash")}),
	})

	var clusterName = "golang-demo-devus-east-1"

	ec2Instance.UserData().AddCommands(
		jsii.String("sudo yum -y update"),
		jsii.String("sudo yum -y install wget"),
		jsii.String("sudo touch /etc/ecs/ecs.config"),
		jsii.String("sudo amazon-linux-extras disable docker"),
		jsii.String("sudo amazon-linux-extras install -y ecs"),
		jsii.String("echo \"ECS_CLUSTER="+clusterName+"\" >>  /etc/ecs/ecs.config"),
		jsii.String("echo \"ECS_AWSVPC_BLOCK_IMDS=true\" >> /etc/ecs/ecs.config"),
		jsii.String("sudo systemctl enable --now --no-block ecs.service"),
	)
	return stack
}
