package breezeware

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type ContainerCompute interface {
	constructs.Construct
	Cluster() awsecs.ICluster
}

type InstanceClass string

type containerCompute struct {
	constructs.Construct
	cluster           awsecs.ICluster
	loadbalancer      awselasticloadbalancingv2.IApplicationLoadBalancer
	cloudmapNamespace awsservicediscovery.PrivateDnsNamespace
	httpsListener     awselasticloadbalancingv2.ApplicationListener
}

type ContainerComputeClusterProps struct {
	ClusterName       string
	ContainerInsights bool
}

type ContainerComputeAsgProps struct {
	Name            string
	MinCapacity     float64
	MaxCapacity     float64
	DesiredCapacity float64
	SshKeyName      string
	InstanceClass   awsec2.InstanceClass
	InstanceSize    awsec2.InstanceSize
	Vpc             awsec2.IVpc
}

type ContainerComputeAsgCapacityProviderProps struct {
	CapacityProviderName string
}

type ContainerComputeLoadBalancerProps struct {
	LoadBalancerName       string
	ListenerCertificateArn string
}

type ContainerComputeCloudmapNamespaceProps struct {
	Name        string
	Description string
}

type VpcProps struct {
	VpcId string
}

type ContainerComputeProps struct {
	Vpc                 VpcProps
	Cluster             ContainerComputeClusterProps
	AutoScalingGroup    ContainerComputeAsgProps
	AsgCapacityProvider []ContainerComputeAsgCapacityProviderProps
	SecurityGroup       ContainerComputeAsgProps
	LoadBalancer        ContainerComputeLoadBalancerProps
	CloudmapNamespace   ContainerComputeCloudmapNamespaceProps
}

func NewContainerCompute(scope constructs.Construct, id *string, props *ContainerComputeProps) ContainerCompute {

	this := constructs.NewConstruct(scope, id)

	vpc := lookupVpc(this, id, &VpcProps{})

	cluster := awsecs.NewCluster(this, jsii.String("EcsCluster"), &awsecs.ClusterProps{
		ClusterName:                    jsii.String(props.Cluster.ClusterName),
		ContainerInsights:              jsii.Bool(props.Cluster.ContainerInsights),
		EnableFargateCapacityProviders: jsii.Bool(false),
		Vpc:                            vpc,
	})

	for _, asgCapacityProvider := range props.AsgCapacityProvider {

		autoScalingGroup := createAutoScalingGroup(this, id, &props.AutoScalingGroup, *cluster.ClusterName())

		asgCapacityProvider := awsecs.NewAsgCapacityProvider(this, jsii.String(asgCapacityProvider.CapacityProviderName+"AsgCapacityProvider"), &awsecs.AsgCapacityProviderProps{
			AutoScalingGroup:                   autoScalingGroup,
			EnableManagedScaling:               jsii.Bool(true),
			EnableManagedTerminationProtection: jsii.Bool(false),
			TargetCapacityPercent:              jsii.Number(100),
			CapacityProviderName:               jsii.String(asgCapacityProvider.CapacityProviderName),
			CanContainersAccessInstanceRole:    jsii.Bool(true),
		})

		cluster.AddAsgCapacityProvider(asgCapacityProvider, &awsecs.AddAutoScalingGroupCapacityOptions{})
	}

	loadBalancer := awselasticloadbalancingv2.NewApplicationLoadBalancer(this, jsii.String("LoadBalanerSetup"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		LoadBalancerName: jsii.String(props.LoadBalancer.LoadBalancerName),
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true),
		VpcSubnets:       &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
		IdleTimeout:      awscdk.Duration_Seconds(jsii.Number(120)),
		IpAddressType:    awselasticloadbalancingv2.IpAddressType_IPV4,
		SecurityGroup: createLbSecurityGroup(this, jsii.String(props.LoadBalancer.LoadBalancerName+"SecurityGroup"), jsii.String(props.LoadBalancer.LoadBalancerName+
			"SecurityGroup"), jsii.String("Security group for "+props.LoadBalancer.LoadBalancerName), vpc),
	})

	httpsListener := awselasticloadbalancingv2.NewApplicationListener(this, jsii.String("LoadbalancerHttpsListener"), &awselasticloadbalancingv2.ApplicationListenerProps{
		LoadBalancer: loadBalancer,
		Certificates: &[]awselasticloadbalancingv2.IListenerCertificate{
			awselasticloadbalancingv2.ListenerCertificate_FromArn(jsii.String(props.LoadBalancer.ListenerCertificateArn))},
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
		Name:        jsii.String(props.CloudmapNamespace.Name),
		Description: jsii.String(props.CloudmapNamespace.Description),
		Vpc:         vpc,
	})

	return &containerCompute{this, cluster, loadBalancer, cloudmapNamespace, httpsListener}
}

func (c *containerCompute) Cluster() awsecs.ICluster {
	return c.cluster
}

func (lb *containerCompute) LoadBalancer() awselasticloadbalancingv2.IApplicationLoadBalancer {
	return lb.loadbalancer
}

func (cm *containerCompute) CloudMapNamespace() awsservicediscovery.IPrivateDnsNamespace {
	return cm.cloudmapNamespace
}

func (hl *containerCompute) HttpsListener() awselasticloadbalancingv2.IApplicationListener {
	return hl.httpsListener
}

func lookupVpc(scope constructs.Construct, id *string, props *VpcProps) awsec2.IVpc {
	vpc := awsec2.Vpc_FromLookup(scope, id, &awsec2.VpcLookupOptions{
		VpcId: jsii.String(props.VpcId),
	})
	return vpc
}

func createAsgSecurityGroup(scope constructs.Construct, id *string, name *string, description *string, vpc awsec2.IVpc) awsec2.ISecurityGroup {
	asgSecurityGroup := awsec2.NewSecurityGroup(scope, id, &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: name,
		Description:       description,
	})
	return asgSecurityGroup
}

func createAutoScalingGroup(scope constructs.Construct, id *string, props *ContainerComputeAsgProps, clusterName string) awsautoscaling.IAutoScalingGroup {

	asgPolicyDocument := awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
		Statements: &[]awsiam.PolicyStatement{awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{Effect: awsiam.Effect_ALLOW,
			Actions: &[]*string{
				jsii.String("ec2:AttachVolume"),
				jsii.String("ec2:CreateVolume"),
				jsii.String("ec2:DeleteVolume"),
				jsii.String("ec2:DescribeAvailabilityZones"),
				jsii.String("ec2:DescribeInstances"),
				jsii.String("ec2:DescribeVolumes"),
				jsii.String("ec2:DescribeVolumeAttribute"),
				jsii.String("ec2:DetachVolume"),
				jsii.String("ec2:DescribeVolumeStatus"),
				jsii.String("ec2:ModifyVolumeAttribute"),
				jsii.String("ec2:DescribeTags"),
				jsii.String("ec2:CreateTags"),
			},
			Resources: &[]*string{jsii.String("*")}})},
	})

	role := awsiam.NewRole(scope, jsii.String("IamRole"+props.Name), &awsiam.RoleProps{
		Description:    jsii.String("Iam role for autoscaling group " + props.Name),
		InlinePolicies: &map[string]awsiam.PolicyDocument{"Ec2VolumeAccess": asgPolicyDocument},
		RoleName:       jsii.String(props.Name + "InstanceProfileRole"),
		AssumedBy:      awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
	})

	image := awsec2.NewAmazonLinuxImage(&awsec2.AmazonLinuxImageProps{
		CpuType:        awsec2.AmazonLinuxCpuType_X86_64,
		Edition:        awsec2.AmazonLinuxEdition_STANDARD,
		Generation:     awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		Virtualization: awsec2.AmazonLinuxVirt_HVM,
		Kernel:         awsec2.AmazonLinuxKernel_KERNEL5_X,
	})

	asg := awsautoscaling.NewAutoScalingGroup(scope, id, &awsautoscaling.AutoScalingGroupProps{
		AutoScalingGroupName: jsii.String(props.Name),
		MinCapacity:          jsii.Number(props.MinCapacity),
		MaxCapacity:          jsii.Number(props.MaxCapacity),
		InstanceType:         awsec2.InstanceType_Of(props.InstanceClass, props.InstanceSize),
		MachineImage:         image,
		SecurityGroup: createAsgSecurityGroup(scope, jsii.String(props.Name+"SecurityGroup"), jsii.String(props.Name+
			"SecurityGroup"), jsii.String("Security group for "+props.Name), props.Vpc),
		UserData:   awsec2.UserData_ForLinux(&awsec2.LinuxUserDataOptions{Shebang: jsii.String("#!/bin/bash")}),
		VpcSubnets: &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
		Vpc:        props.Vpc,
		KeyName:    jsii.String(props.SshKeyName),
		Role:       role,
	})

	asg.UserData().AddCommands(
		jsii.String("sudo yum -y update"),
		jsii.String("sudo yum -y install wget"),
		jsii.String("sudo touch /etc/ecs/ecs.config"),
		jsii.String("sudo amazon-linux-extras disable docker"),
		jsii.String("sudo amazon-linux-extras install -y ecs"),
		jsii.String("echo \"ECS_CLUSTER="+clusterName+"\" >>  /etc/ecs/ecs.config"),
		jsii.String("echo \"ECS_AWSVPC_BLOCK_IMDS=true\" >> /etc/ecs/ecs.config"),
		jsii.String("sudo systemctl enable --now --no-block ecs.service"),
		jsii.String("docker plugin install rexray/ebs REXRAY_PREEMPT=true EBS_REGION="+*awscdk.Aws_REGION()+" --grant-all-permissions"),
	)
	return asg
}

func createLbSecurityGroup(scope constructs.Construct, id *string, name *string, description *string, vpc awsec2.IVpc) awsec2.ISecurityGroup {
	lbSecurityGroup := awsec2.NewSecurityGroup(scope, id, &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: name,
		Description:       description,
	})

	lbSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(443)),
		jsii.String("Default HTTPS Port"),
		jsii.Bool(false),
	)

	lbSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Default HTTP Port"),
		jsii.Bool(false),
	)

	return lbSecurityGroup
}
