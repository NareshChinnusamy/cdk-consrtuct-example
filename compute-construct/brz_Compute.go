package breezeware

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
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
	cluster           awsecs.ICluster
	loadbalancer      awselasticloadbalancingv2.ApplicationLoadBalancer
	cloudmapNamespace awsservicediscovery.PrivateDnsNamespace
	httpsListener     awselasticloadbalancingv2.ApplicationListener
}

type ContainerComputeClusterProps struct {
	ClusterName       *string
	ContainerInsights *bool
}

type ContainerComputeSecurityGroupProps struct {
	AsgGroupName                 *string
	AsgGroupDescription          *string
	LoadbalancerGroupName        *string
	LoadbalancerGroupDescription *string
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

type ContainerComputeLoadBalancerProps struct {
	LoadBalancerName       *string
	ListenerCertificateArn *string
}

type ContainerComputeCloudmapNamespaceProps struct {
	Name        *string
	Description *string
}

type ContainerComputeProps struct {
	VpcId                    *string
	ClusterProps             *ContainerComputeClusterProps
	AsgCapacityProviderProps []*ContainerComputeAsgCapacityProviderProps
	SecurityGroupProps       *ContainerComputeSecurityGroupProps
	LoadBalancerProps        *ContainerComputeLoadBalancerProps
	CloudmapNamespaceProps   *ContainerComputeCloudmapNamespaceProps
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
	asgSecurityGroup := awsec2.NewSecurityGroup(this, jsii.String("SecurityGroup"), &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: props.SecurityGroupProps.AsgGroupName,
		Description:       props.SecurityGroupProps.AsgGroupDescription,
	})

	loadBalancerSecurityGroup := awsec2.NewSecurityGroup(this, jsii.String("SecurityGroup"), &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: props.SecurityGroupProps.LoadbalancerGroupName,
		Description:       props.SecurityGroupProps.LoadbalancerGroupDescription,
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
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Default HTTP Port"),
		jsii.Bool(false),
	)

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
			SecurityGroup:        asgSecurityGroup,
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

	loadBalancer := awselasticloadbalancingv2.NewApplicationLoadBalancer(this, jsii.String("LoadBalanerSetup"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		LoadBalancerName: props.LoadBalancerProps.LoadBalancerName,
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true),
		VpcSubnets:       &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
		IdleTimeout:      awscdk.Duration_Seconds(jsii.Number(120)),
		IpAddressType:    awselasticloadbalancingv2.IpAddressType_IPV4,
		SecurityGroup:    loadBalancerSecurityGroup,
	})

	httpsListener := awselasticloadbalancingv2.NewApplicationListener(this, jsii.String("LoadbalancerHttpdListener"), &awselasticloadbalancingv2.ApplicationListenerProps{
		LoadBalancer: loadBalancer,
		Certificates: &[]awselasticloadbalancingv2.IListenerCertificate{
			awselasticloadbalancingv2.ListenerCertificate_FromArn(props.LoadBalancerProps.ListenerCertificateArn)},
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
		Port:     jsii.Number(443),
		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
			awselasticloadbalancingv2.NewApplicationTargetGroup(
				this,
				jsii.String("DefaultTargetGroup"),
				&awselasticloadbalancingv2.ApplicationTargetGroupProps{
					TargetGroupName: jsii.String("DefaultTargetGroup"),
					TargetType:      awselasticloadbalancingv2.TargetType_INSTANCE,
					Vpc:             vpc,
					Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
					Port:            jsii.Number(8080),
				},
			),
		},
	})

	awselasticloadbalancingv2.NewApplicationListener(this, jsii.String("LoadbalancerHttpListener"), &awselasticloadbalancingv2.ApplicationListenerProps{
		Port:         jsii.Number(80),
		LoadBalancer: loadBalancer,
		DefaultAction: awselasticloadbalancingv2.ListenerAction_Redirect(
			&awselasticloadbalancingv2.RedirectOptions{
				Host:      jsii.String("#{host}"),
				Protocol:  jsii.String("HTTPS"),
				Port:      jsii.String("443"),
				Path:      jsii.String("/#{path}"),
				Query:     jsii.String("#{query}"),
				Permanent: jsii.Bool(true),
			}),
	})

	cloudmapNamespace := awsservicediscovery.NewPrivateDnsNamespace(this, jsii.String("CloudMapNamespace"), &awsservicediscovery.PrivateDnsNamespaceProps{
		Name:        props.CloudmapNamespaceProps.Name,
		Description: props.CloudmapNamespaceProps.Description,
		Vpc:         vpc,
	})

	return &containerCompute{this, cluster, loadBalancer, cloudmapNamespace, httpsListener}
}

func (c *containerCompute) Cluster() awsecs.ICluster {
	return c.cluster
}
