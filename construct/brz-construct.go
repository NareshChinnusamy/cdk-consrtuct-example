package BrzConstruct

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type BrzCustomProps struct {
	S3Props  awss3.BucketProps
	Ec2Props awsec2.InstanceProps
}

type brzStruct struct {
	constructs.Construct
	bucket awss3.Bucket
	ec2    awsec2.Instance
}

func BrzL3Construct(scope constructs.Construct, id string, props *BrzCustomProps) brzStruct {
	this := constructs.NewConstruct(scope, &id)

	s3 := awss3.NewBucket(this, jsii.String("BucketBrz"), &props.S3Props)

	ec2Instance := awsec2.NewInstance(this, jsii.String("InstanceBrz"), &props.Ec2Props)
	return brzStruct{this, s3, ec2Instance}
}
