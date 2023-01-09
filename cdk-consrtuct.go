package main

import (
	BrzConstruct "github.com/NareshChinnusamy/go-module"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkConsrtuctStackProps struct {
	awscdk.StackProps
}

func NewCdkConsrtuctStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
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

	BrzConstruct.BrzL3Construct(stack, "DemoBrzStack",
		&BrzConstruct.BrzCustomProps{S3Props: awss3.BucketProps{BucketName: jsii.String("brz-demo-s3-bucket-" + regionName)},
			Ec2Props: awsec2.InstanceProps{Vpc: vpc, PropagateTagsToVolumeOnCreation: jsii.Bool(true),
				InstanceType: awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE2, awsec2.InstanceSize_MICRO),
				MachineImage: image, KeyName: jsii.String("breezethru-demo-key-pair"), SecurityGroup: securityGroup, InstanceName: jsii.String("brz-demo-" + regionName)}})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewCdkConsrtuctStack(app, "BrzDemoStack", &CdkConsrtuctStackProps{
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
