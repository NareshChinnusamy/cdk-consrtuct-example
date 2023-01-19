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
	LoadBalancer() awselasticloadbalancingv2.IApplicationLoadBalancer
	CloudMapNamespace() awsservicediscovery.IPrivateDnsNamespace
	HttpsListener() awselasticloadbalancingv2.IApplicationListener
}

type containerCompute struct {
	constructs.Construct
	cluster           awsecs.ICluster
	loadbalancer      awselasticloadbalancingv2.IApplicationLoadBalancer
	cloudmapNamespace awsservicediscovery.IPrivateDnsNamespace
	httpsListener     awselasticloadbalancingv2.IApplicationListener
}

type VpcProps struct {
	VpcId string
}

type ContainerComputeClusterProps struct {
	Name                             string
	ContainerInsights                bool
	IsAsgCapacityProviderEnabled     bool
	IsFargateCapacityProviderEnabled bool
	Vpc                              awsec2.IVpc
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
	Name string
}

type ContainerComputeLoadBalancerProps struct {
	Name                   string
	ListenerCertificateArn string
	Vpc                    awsec2.IVpc
}

type ContainerComputeCloudmapNamespaceProps struct {
	Name        string
	Description string
	Vpc         awsec2.IVpc
}

type securityGroupProps struct {
	Name        string
	Description string
	Vpc         awsec2.IVpc
}

type AutoscalinGroupCapacityProviders struct {
	AutoScalingGroup ContainerComputeAsgProps
	CapacityProvider ContainerComputeAsgCapacityProviderProps
}

type ContainerComputeProps struct {
	Cluster              ContainerComputeClusterProps
	AsgCapacityProviders []AutoscalinGroupCapacityProviders
	LoadBalancer         ContainerComputeLoadBalancerProps
	CloudmapNamespace    ContainerComputeCloudmapNamespaceProps
}

func NewContainerCompute(scope constructs.Construct, id *string, props *ContainerComputeProps) ContainerCompute {

	this := constructs.NewConstruct(scope, id)

	cluster := createCluster(this, jsii.String("EcsCluster"), &props.Cluster)

	if props.Cluster.IsAsgCapacityProviderEnabled {
		for _, asgCapacityProvider := range props.AsgCapacityProviders {

			autoScalingGroup := createAutoScalingGroup(this, jsii.String(asgCapacityProvider.AutoScalingGroup.Name+"AutoscalingGroup"), &asgCapacityProvider.AutoScalingGroup, *cluster.ClusterName())

			capacityProvider := createCapacityProvider(this, jsii.String(asgCapacityProvider.CapacityProvider.Name+"AsgCapacityProvider"), &asgCapacityProvider.CapacityProvider, autoScalingGroup)

			cluster.AddAsgCapacityProvider(capacityProvider, &awsecs.AddAutoScalingGroupCapacityOptions{})
		}
	}
	loadBalancer := createLoadBalancer(this, jsii.String("LoadBalanerSetup"), &props.LoadBalancer)

	httpsListener := createHttpsListener(this, jsii.String("HttpsListener"), &props.LoadBalancer, loadBalancer)

	createHttpListener(this, jsii.String("HttpListener"), loadBalancer)

	cloudmapNamespace := createCloudMapNamespace(this, jsii.String("CloudMapNamespace"), &props.CloudmapNamespace)

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

func LookupVpc(scope constructs.Construct, id *string, props *VpcProps) awsec2.IVpc {
	vpc := awsec2.Vpc_FromLookup(scope, id, &awsec2.VpcLookupOptions{
		VpcId: jsii.String(props.VpcId),
	})
	return vpc
}

func createCluster(scope constructs.Construct, id *string, props *ContainerComputeClusterProps) awsecs.Cluster {
	if props.IsFargateCapacityProviderEnabled {
		cluster := awsecs.NewCluster(scope, id, &awsecs.ClusterProps{
			ClusterName:                    jsii.String(props.Name),
			ContainerInsights:              jsii.Bool(props.ContainerInsights),
			EnableFargateCapacityProviders: jsii.Bool(true),
			Vpc:                            props.Vpc,
		})
		return cluster
	} else {
		cluster := awsecs.NewCluster(scope, id, &awsecs.ClusterProps{
			ClusterName:                    jsii.String(props.Name),
			ContainerInsights:              jsii.Bool(props.ContainerInsights),
			EnableFargateCapacityProviders: jsii.Bool(false),
			Vpc:                            props.Vpc,
		})
		return cluster
	}
}

func createLbSecurityGroup(scope constructs.Construct, id *string, props *securityGroupProps, vpc awsec2.IVpc) awsec2.ISecurityGroup {
	lbSecurityGroup := awsec2.NewSecurityGroup(scope, id, &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: &props.Name,
		Description:       &props.Description,
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

func createLoadBalancer(scope constructs.Construct, id *string, props *ContainerComputeLoadBalancerProps) awselasticloadbalancingv2.IApplicationLoadBalancer {
	lb := awselasticloadbalancingv2.NewApplicationLoadBalancer(scope, id, &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		LoadBalancerName: jsii.String(props.Name),
		Vpc:              props.Vpc,
		InternetFacing:   jsii.Bool(true),
		VpcSubnets:       &awsec2.SubnetSelection{SubnetType: awsec2.SubnetType_PUBLIC},
		IdleTimeout:      awscdk.Duration_Seconds(jsii.Number(120)),
		IpAddressType:    awselasticloadbalancingv2.IpAddressType_IPV4,
		SecurityGroup: createLbSecurityGroup(scope, jsii.String(props.Name+"SecurityGroup"), &securityGroupProps{
			Name:        props.Name + "SecurityGroup",
			Description: "Security group for " + props.Name,
		},
			props.Vpc,
		),
	})
	return lb
}

func createHttpsListener(scope constructs.Construct, id *string, props *ContainerComputeLoadBalancerProps, lb awselasticloadbalancingv2.IApplicationLoadBalancer) awselasticloadbalancingv2.IApplicationListener {
	httpsListener := awselasticloadbalancingv2.NewApplicationListener(scope, jsii.String("LoadbalancerHttpsListener"), &awselasticloadbalancingv2.ApplicationListenerProps{
		LoadBalancer: lb,
		Certificates: &[]awselasticloadbalancingv2.IListenerCertificate{
			awselasticloadbalancingv2.ListenerCertificate_FromArn(jsii.String(props.ListenerCertificateArn))},
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
		Port:     jsii.Number(443),
		DefaultTargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
			awselasticloadbalancingv2.NewApplicationTargetGroup(
				scope,
				jsii.String("DefaultTargetGroup"),
				&awselasticloadbalancingv2.ApplicationTargetGroupProps{
					TargetGroupName: jsii.String(props.Name + "DefaultTargetGroup"),
					TargetType:      awselasticloadbalancingv2.TargetType_INSTANCE,
					Vpc:             props.Vpc,
					Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
					Port:            jsii.Number(8080),
				},
			),
		},
	})
	return httpsListener
}

func createHttpListener(scope constructs.Construct, id *string, lb awselasticloadbalancingv2.IApplicationLoadBalancer) {

	awselasticloadbalancingv2.NewApplicationListener(scope, jsii.String("LoadbalancerHttpListener"), &awselasticloadbalancingv2.ApplicationListenerProps{
		Port:         jsii.Number(80),
		LoadBalancer: lb,
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
}

func createCloudMapNamespace(scope constructs.Construct, id *string, props *ContainerComputeCloudmapNamespaceProps) awsservicediscovery.IPrivateDnsNamespace {
	cloudmapNamespace := awsservicediscovery.NewPrivateDnsNamespace(scope, id, &awsservicediscovery.PrivateDnsNamespaceProps{
		Name:        jsii.String(props.Name),
		Description: jsii.String(props.Description),
		Vpc:         props.Vpc,
	})
	return cloudmapNamespace
}

func createAsgSecurityGroup(scope constructs.Construct, id *string, props *securityGroupProps) awsec2.ISecurityGroup {
	asgSecurityGroup := awsec2.NewSecurityGroup(scope, id, &awsec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               props.Vpc,
		SecurityGroupName: &props.Name,
		Description:       &props.Description,
	})
	return asgSecurityGroup
}

func createAsgPolicyDocument() awsiam.PolicyDocument {
	pd := awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
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
	return pd
}

func createAsgRole(scope constructs.Construct, id *string, props *ContainerComputeAsgProps, policyDocument awsiam.PolicyDocument) awsiam.IRole {
	role := awsiam.NewRole(scope, id, &awsiam.RoleProps{
		Description:    jsii.String("Iam role for autoscaling group " + props.Name),
		InlinePolicies: &map[string]awsiam.PolicyDocument{"Ec2VolumeAccess": policyDocument},
		RoleName:       jsii.String(props.Name + "InstanceProfileRole"),
		AssumedBy:      awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
	})
	return role
}

func createAutoScalingGroup(scope constructs.Construct, id *string, props *ContainerComputeAsgProps, clusterName string) awsautoscaling.IAutoScalingGroup {
	asgPolicyDocument := createAsgPolicyDocument()

	role := createAsgRole(scope, jsii.String("IamRole"+props.Name), props, asgPolicyDocument)

	asg := awsautoscaling.NewAutoScalingGroup(scope, id, &awsautoscaling.AutoScalingGroupProps{
		AutoScalingGroupName: jsii.String(props.Name),
		MinCapacity:          jsii.Number(props.MinCapacity),
		MaxCapacity:          jsii.Number(props.MaxCapacity),
		InstanceType:         awsec2.InstanceType_Of(props.InstanceClass, props.InstanceSize),
		MachineImage:         createMachineImage(),
		SecurityGroup: createAsgSecurityGroup(scope, jsii.String(props.Name+"SecurityGroup"), &securityGroupProps{
			Name:        props.Name + "SecurityGroup",
			Description: "SecurityGroup for " + props.Name,
			Vpc:         props.Vpc,
		}),

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

func createMachineImage() awsec2.IMachineImage {
	image := awsec2.NewAmazonLinuxImage(&awsec2.AmazonLinuxImageProps{
		CpuType:        awsec2.AmazonLinuxCpuType_X86_64,
		Edition:        awsec2.AmazonLinuxEdition_STANDARD,
		Generation:     awsec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		Virtualization: awsec2.AmazonLinuxVirt_HVM,
		Kernel:         awsec2.AmazonLinuxKernel_KERNEL5_X,
	})
	return image
}

func createCapacityProvider(scope constructs.Construct, id *string, props *ContainerComputeAsgCapacityProviderProps, asg awsautoscaling.IAutoScalingGroup) awsecs.AsgCapacityProvider {
	asgCapacityProvider := awsecs.NewAsgCapacityProvider(scope, id, &awsecs.AsgCapacityProviderProps{
		AutoScalingGroup:                   asg,
		EnableManagedScaling:               jsii.Bool(true),
		EnableManagedTerminationProtection: jsii.Bool(false),
		TargetCapacityPercent:              jsii.Number(100),
		CapacityProviderName:               jsii.String(props.Name),
		CanContainersAccessInstanceRole:    jsii.Bool(true),
	})
	return asgCapacityProvider
}
