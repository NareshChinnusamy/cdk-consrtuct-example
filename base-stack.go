package main

import (
	construct "cdk-consrtuct/construct"

	clusterConstruct "cdk-consrtuct/compute-construct"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkConsrtuctStackProps struct {
	awscdk.StackProps
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

	nestedStack := awscdk.NewNestedStack(stack, jsii.String("NestedStack"), &awscdk.NestedStackProps{Description: jsii.String("NestedStackresource")})

	vpc := awsec2.Vpc_FromLookup(stack, &vpcName, &awsec2.VpcLookupOptions{VpcId: &vpcId})

	image := awsec2.NewAmazonLinuxImage(&awsec2.AmazonLinuxImageProps{CpuType: awsec2.AmazonLinuxCpuType_X86_64, Edition: awsec2.AmazonLinuxEdition_STANDARD,
		Generation: awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2, Virtualization: awsec2.AmazonLinuxVirt_HVM, Kernel: awsec2.AmazonLinuxKernel_KERNEL5_X})

	securityGroup := awsec2.SecurityGroup_FromLookupById(stack, jsii.String(sgName), jsii.String(sgId))

	construct.BrzL3Construct(stack, "DemoBrzStack",
		&construct.BrzCustomProps{S3Props: awss3.BucketProps{BucketName: jsii.String("brz-demo-s3-bucket-" + regionName)},
			Ec2Props: awsec2.InstanceProps{Vpc: vpc, PropagateTagsToVolumeOnCreation: jsii.Bool(true),
				InstanceType: awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE2, awsec2.InstanceSize_MICRO),
				MachineImage: image, KeyName: jsii.String("breezethru-demo-key-pair"), SecurityGroup: securityGroup, InstanceName: jsii.String("brz-demo-" + regionName)}})

	awselasticloadbalancingv2.NewApplicationTargetGroup(nestedStack, jsii.String("TargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{TargetType: awselasticloadbalancingv2.TargetType_IP,
		TargetGroupName: jsii.String("DemoTg"), Vpc: vpc, Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			Enabled:          jsii.Bool(true),
			HealthyHttpCodes: jsii.String("200"),
			Path:             jsii.String("/"),
			Interval:         awscdk.Duration_Seconds(jsii.Number(30))}})

	return stack
}

func clusterStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	clusterConstruct.NewContainerCompute(stack, "Cluster", &clusterConstruct.ContainerComputeProps{
		AsgCapacityProviderProps: []*clusterConstruct.ContainerComputeAsgCapacityProviderProps{{
			AutoScalingGroupName: jsii.String("Demo"),
			InstanceClass:        awsec2.InstanceClass_BURSTABLE2,
			InstanceSize:         awsec2.InstanceSize_MEDIUM,
			MinCapacity:          jsii.Number(0),
			MaxCapacity:          jsii.Number(2),
			SshKeyName:           jsii.String("demo-key-pair"),
			UserData:             jsii.String("demo-ssh"),
		}, {
			AutoScalingGroupName: jsii.String("Demo"),
			InstanceClass:        awsec2.InstanceClass_BURSTABLE2,
			InstanceSize:         awsec2.InstanceSize_MEDIUM,
			MinCapacity:          jsii.Number(0),
			MaxCapacity:          jsii.Number(2),
			SshKeyName:           jsii.String("demo-key-pair"),
			UserData:             jsii.String("demo-ssh"),
		}}})
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

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	CdkConsrtuctStack(app, "BrzDemoStack", &CdkConsrtuctStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String("305251478828"),
		Region:  jsii.String("us-east-1"),
	}
}
