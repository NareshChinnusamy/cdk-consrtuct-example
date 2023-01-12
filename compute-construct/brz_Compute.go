package breezeware

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type ContainerCompute interface {
	constructs.Construct
	Cluster() awsecs.ICluster
}

type InstanceClass string

const InstanceClass_BURSTABLE2 InstanceClass = "BURSTABLE2"

type containerCompute struct {
	constructs.Construct
	cluster awsecs.ICluster
}

type ContainerComputeClusterProps struct {
	ClusterName       *string
	ContainerInsights *bool
}

type ContainerComputeSecurityGroupProps struct {
	GroupName        *string
	GroupDescription *string
}

type ContainerComputeAsgCapacityProviderProps struct {
	AutoScalingGroupName *string
	MinCapacity          *float64
	MaxCapacity          *float64
	SshKeyName           *string
	UserData             *string
	InstanceClass        awsec2.InstanceClass
	InstanceSize         awsec2.InstanceSize
	CapacityProviderName *string
}

type ContainerComputeProps struct {
	VpcId                    *string
	ClusterProps             *ContainerComputeClusterProps
	AsgCapacityProviderProps []*ContainerComputeAsgCapacityProviderProps
	SecurityGroupProps       *ContainerComputeSecurityGroupProps
}

func NewContainerCompute(scope constructs.Construct, id string, props *ContainerComputeProps) ContainerCompute {

	this := constructs.NewConstruct(scope, &id)

	vpc := awsec2.Vpc_FromLookup(this, jsii.String("LookUpVpc"), &awsec2.VpcLookupOptions{
		VpcId: props.VpcId,
	})

	image := awsec2.NewAmazonLinuxImage(&awsec2.AmazonLinuxImageProps{
		CpuType:        awsec2.AmazonLinuxCpuType_X86_64,
		Edition:        awsec2.AmazonLinuxEdition_STANDARD,
		Generation:     awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		Virtualization: awsec2.AmazonLinuxVirt_HVM,
		Kernel:         awsec2.AmazonLinuxKernel_KERNEL5_X,
	})

	//Todo
	securityGroup := awsec2.NewSecurityGroup(this, jsii.String("SecurityGroup"), &awsec2.SecurityGroupProps{
		AllowAllOutbound: jsii.Bool(true),
		Vpc:              vpc, SecurityGroupName: props.SecurityGroupProps.GroupName,
		Description: props.SecurityGroupProps.GroupDescription,
	})

	cluster := awsecs.NewCluster(this, jsii.String("EcsCluster"), &awsecs.ClusterProps{
		ClusterName:                    props.ClusterProps.ClusterName,
		ContainerInsights:              props.ClusterProps.ContainerInsights,
		EnableFargateCapacityProviders: jsii.Bool(false),
		Vpc:                            vpc,
	})

	for _, asgCapacityProvider := range props.AsgCapacityProviderProps {
		autoScalingGroup := awsautoscaling.NewAutoScalingGroup(this, jsii.String(*asgCapacityProvider.AutoScalingGroupName+"AutoScalingGroup"), &awsautoscaling.AutoScalingGroupProps{
			AutoScalingGroupName: asgCapacityProvider.AutoScalingGroupName,
			MinCapacity:          asgCapacityProvider.MaxCapacity,
			MaxCapacity:          asgCapacityProvider.MaxCapacity,
			InstanceType:         awsec2.InstanceType_Of(asgCapacityProvider.InstanceClass, asgCapacityProvider.InstanceSize),
			MachineImage:         image,
			SecurityGroup:        securityGroup,
			UserData:             awsec2.UserData_ForLinux(&awsec2.LinuxUserDataOptions{Shebang: asgCapacityProvider.UserData}),
			VpcSubnets:           &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
			Vpc:                  vpc,
			KeyName:              asgCapacityProvider.SshKeyName,
		})

		asgCapacityProvider := awsecs.NewAsgCapacityProvider(this, jsii.String(*asgCapacityProvider.AutoScalingGroupName+"AsgCapacityProvider"), &awsecs.AsgCapacityProviderProps{
			AutoScalingGroup:                   autoScalingGroup,
			EnableManagedScaling:               jsii.Bool(true),
			EnableManagedTerminationProtection: jsii.Bool(false),
			TargetCapacityPercent:              jsii.Number(100),
			CapacityProviderName:               asgCapacityProvider.CapacityProviderName,
			CanContainersAccessInstanceRole:    jsii.Bool(true),
		})

		cluster.AddAsgCapacityProvider(asgCapacityProvider, &awsecs.AddAutoScalingGroupCapacityOptions{})
	}

	return &containerCompute{this, cluster}
}

func (c *containerCompute) Cluster() awsecs.ICluster {
	return c.cluster
}
