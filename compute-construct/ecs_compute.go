package breezeware

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	autoscaling "github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	ecs "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	elbv2 "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	servicediscovery "github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

var vpc ec2.IVpc

type ContainerCompute interface {
	constructs.Construct
	Cluster() ecs.ICluster
	LoadBalancer() elbv2.IApplicationLoadBalancer
	CloudMapNamespace() servicediscovery.IPrivateDnsNamespace
	HttpsListener() elbv2.IApplicationListener
}

type containerCompute struct {
	constructs.Construct
	cluster           ecs.ICluster
	loadbalancer      elbv2.IApplicationLoadBalancer
	cloudmapNamespace servicediscovery.IPrivateDnsNamespace
	httpsListener     elbv2.IApplicationListener
}

type VpcProps struct {
	VpcId string
}

type ContainerComputeClusterProps struct {
	Name                             string
	ContainerInsights                bool
	IsAsgCapacityProviderEnabled     bool
	IsFargateCapacityProviderEnabled bool
	vpc                              ec2.IVpc
}

type ContainerComputeAsgProps struct {
	Name            string
	MinCapacity     float64
	MaxCapacity     float64
	DesiredCapacity float64
	SshKeyName      string
	InstanceClass   ec2.InstanceClass
	InstanceSize    ec2.InstanceSize
	vpc             ec2.IVpc
}

type ContainerComputeAsgCapacityProviderProps struct {
	Name string
}

type ContainerComputeLoadBalancerProps struct {
	Name                   string
	ListenerCertificateArn string
	vpc                    ec2.IVpc
}

type ContainerComputeCloudmapNamespaceProps struct {
	Name        string
	Description string
	vpc         ec2.IVpc
}

type securityGroupProps struct {
	Name        string
	Description string
	vpc         ec2.IVpc
}

type AutoscalinGroupCapacityProviders struct {
	AutoScalingGroup ContainerComputeAsgProps
	CapacityProvider ContainerComputeAsgCapacityProviderProps
}

type ContainerComputeProps struct {
	VpcId                *string
	Cluster              ContainerComputeClusterProps
	AsgCapacityProviders []AutoscalinGroupCapacityProviders
	LoadBalancer         ContainerComputeLoadBalancerProps
	CloudmapNamespace    ContainerComputeCloudmapNamespaceProps
}

func NewContainerCompute(scope constructs.Construct, id *string, props *ContainerComputeProps) ContainerCompute {

	this := constructs.NewConstruct(scope, id)

	vpc = LookupVpc(scope, jsii.String("LookUpVpc"), &VpcProps{VpcId: *props.VpcId})

	cluster := createCluster(this, jsii.String("EcsCluster"), &props.Cluster)

	if props.Cluster.IsAsgCapacityProviderEnabled {
		for _, asgCapacityProvider := range props.AsgCapacityProviders {

			autoScalingGroup := createAutoScalingGroup(this, jsii.String(asgCapacityProvider.AutoScalingGroup.Name+"AutoscalingGroup"), &asgCapacityProvider.AutoScalingGroup, *cluster.ClusterName())

			capacityProvider := createCapacityProvider(this, jsii.String(asgCapacityProvider.CapacityProvider.Name+"AsgCapacityProvider"), &asgCapacityProvider.CapacityProvider, autoScalingGroup)

			cluster.AddAsgCapacityProvider(capacityProvider, &ecs.AddAutoScalingGroupCapacityOptions{})
		}
	}
	loadBalancer := createLoadBalancer(this, jsii.String("LoadBalanerSetup"), &props.LoadBalancer)

	httpsListener := createHttpsListener(this, jsii.String("HttpsListener"), &props.LoadBalancer, loadBalancer)

	createHttpListener(this, jsii.String("HttpListener"), loadBalancer)

	cloudmapNamespace := createCloudMapNamespace(this, jsii.String("CloudMapNamespace"), &props.CloudmapNamespace)

	return &containerCompute{this, cluster, loadBalancer, cloudmapNamespace, httpsListener}
}

func (c *containerCompute) Cluster() ecs.ICluster {
	return c.cluster
}

func (lb *containerCompute) LoadBalancer() elbv2.IApplicationLoadBalancer {
	return lb.loadbalancer
}

func (cm *containerCompute) CloudMapNamespace() servicediscovery.IPrivateDnsNamespace {
	return cm.cloudmapNamespace
}

func (hl *containerCompute) HttpsListener() elbv2.IApplicationListener {
	return hl.httpsListener
}

func LookupVpc(scope constructs.Construct, id *string, props *VpcProps) ec2.IVpc {
	vpc := ec2.Vpc_FromLookup(scope, id, &ec2.VpcLookupOptions{
		VpcId: jsii.String(props.VpcId),
	})
	return vpc
}

func createCluster(scope constructs.Construct, id *string, props *ContainerComputeClusterProps) ecs.Cluster {
	if props.IsFargateCapacityProviderEnabled {
		cluster := ecs.NewCluster(scope, id, &ecs.ClusterProps{
			ClusterName:                    jsii.String(props.Name),
			ContainerInsights:              jsii.Bool(props.ContainerInsights),
			EnableFargateCapacityProviders: jsii.Bool(true),
			Vpc:                            vpc,
		})
		return cluster
	} else {
		cluster := ecs.NewCluster(scope, id, &ecs.ClusterProps{
			ClusterName:                    jsii.String(props.Name),
			ContainerInsights:              jsii.Bool(props.ContainerInsights),
			EnableFargateCapacityProviders: jsii.Bool(false),
			Vpc:                            vpc,
		})
		return cluster
	}
}

func createLbSecurityGroup(scope constructs.Construct, id *string, props *securityGroupProps, vpc ec2.IVpc) ec2.ISecurityGroup {
	lbSecurityGroup := ec2.NewSecurityGroup(scope, id, &ec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: &props.Name,
		Description:       &props.Description,
	})

	lbSecurityGroup.AddIngressRule(
		ec2.Peer_AnyIpv4(),
		ec2.Port_Tcp(jsii.Number(443)),
		jsii.String("Default HTTPS Port"),
		jsii.Bool(false),
	)

	lbSecurityGroup.AddIngressRule(
		ec2.Peer_AnyIpv4(),
		ec2.Port_Tcp(jsii.Number(80)),
		jsii.String("Default HTTP Port"),
		jsii.Bool(false),
	)

	return lbSecurityGroup
}

func createLoadBalancer(scope constructs.Construct, id *string, props *ContainerComputeLoadBalancerProps) elbv2.IApplicationLoadBalancer {
	lb := elbv2.NewApplicationLoadBalancer(scope, id, &elbv2.ApplicationLoadBalancerProps{
		LoadBalancerName: jsii.String(props.Name),
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true),
		VpcSubnets:       &ec2.SubnetSelection{SubnetType: ec2.SubnetType_PUBLIC},
		IdleTimeout:      awscdk.Duration_Seconds(jsii.Number(120)),
		IpAddressType:    elbv2.IpAddressType_IPV4,
		SecurityGroup: createLbSecurityGroup(scope, jsii.String(props.Name+"SecurityGroup"), &securityGroupProps{
			Name:        props.Name + "SecurityGroup",
			Description: "Security group for " + props.Name,
		},
			vpc,
		),
	})
	return lb
}

func createHttpsListener(scope constructs.Construct, id *string, props *ContainerComputeLoadBalancerProps, lb elbv2.IApplicationLoadBalancer) elbv2.IApplicationListener {
	httpsListener := elbv2.NewApplicationListener(scope, jsii.String("LoadbalancerHttpsListener"), &elbv2.ApplicationListenerProps{
		LoadBalancer: lb,
		Certificates: &[]elbv2.IListenerCertificate{
			elbv2.ListenerCertificate_FromArn(jsii.String(props.ListenerCertificateArn))},
		Protocol: elbv2.ApplicationProtocol_HTTPS,
		Port:     jsii.Number(443),
		DefaultTargetGroups: &[]elbv2.IApplicationTargetGroup{
			elbv2.NewApplicationTargetGroup(
				scope,
				jsii.String("DefaultTargetGroup"),
				&elbv2.ApplicationTargetGroupProps{
					TargetGroupName: jsii.String(props.Name + "DefaultTargetGroup"),
					TargetType:      elbv2.TargetType_INSTANCE,
					Vpc:             vpc,
					Protocol:        elbv2.ApplicationProtocol_HTTP,
					Port:            jsii.Number(8080),
				},
			),
		},
	})
	return httpsListener
}

func createHttpListener(scope constructs.Construct, id *string, lb elbv2.IApplicationLoadBalancer) {

	elbv2.NewApplicationListener(scope, jsii.String("LoadbalancerHttpListener"), &elbv2.ApplicationListenerProps{
		Port:         jsii.Number(80),
		LoadBalancer: lb,
		DefaultAction: elbv2.ListenerAction_Redirect(
			&elbv2.RedirectOptions{
				Host:      jsii.String("#{host}"),
				Protocol:  jsii.String("HTTPS"),
				Port:      jsii.String("443"),
				Path:      jsii.String("/#{path}"),
				Query:     jsii.String("#{query}"),
				Permanent: jsii.Bool(true),
			}),
	})
}

func createCloudMapNamespace(scope constructs.Construct, id *string, props *ContainerComputeCloudmapNamespaceProps) servicediscovery.IPrivateDnsNamespace {
	cloudmapNamespace := servicediscovery.NewPrivateDnsNamespace(scope, id, &servicediscovery.PrivateDnsNamespaceProps{
		Name:        jsii.String(props.Name),
		Description: jsii.String(props.Description),
		Vpc:         vpc,
	})
	return cloudmapNamespace
}

func createAsgSecurityGroup(scope constructs.Construct, id *string, props *securityGroupProps) ec2.ISecurityGroup {
	asgSecurityGroup := ec2.NewSecurityGroup(scope, id, &ec2.SecurityGroupProps{
		AllowAllOutbound:  jsii.Bool(true),
		Vpc:               vpc,
		SecurityGroupName: &props.Name,
		Description:       &props.Description,
	})
	return asgSecurityGroup
}

func createAsgPolicyDocument() iam.PolicyDocument {
	pd := iam.NewPolicyDocument(&iam.PolicyDocumentProps{
		Statements: &[]iam.PolicyStatement{iam.NewPolicyStatement(&iam.PolicyStatementProps{Effect: iam.Effect_ALLOW,
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

func createAsgRole(scope constructs.Construct, id *string, props *ContainerComputeAsgProps, policyDocument iam.PolicyDocument) iam.IRole {
	role := iam.NewRole(scope, id, &iam.RoleProps{
		Description:    jsii.String("Iam role for autoscaling group " + props.Name),
		InlinePolicies: &map[string]iam.PolicyDocument{"Ec2VolumeAccess": policyDocument},
		RoleName:       jsii.String(props.Name + "InstanceProfileRole"),
		AssumedBy:      iam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &iam.ServicePrincipalOpts{}),
	})
	return role
}

func createAutoScalingGroup(scope constructs.Construct, id *string, props *ContainerComputeAsgProps, clusterName string) autoscaling.IAutoScalingGroup {
	asgPolicyDocument := createAsgPolicyDocument()

	role := createAsgRole(scope, jsii.String("IamRole"+props.Name), props, asgPolicyDocument)

	asg := autoscaling.NewAutoScalingGroup(scope, id, &autoscaling.AutoScalingGroupProps{
		AutoScalingGroupName: jsii.String(props.Name),
		MinCapacity:          jsii.Number(props.MinCapacity),
		MaxCapacity:          jsii.Number(props.MaxCapacity),
		InstanceType:         ec2.InstanceType_Of(props.InstanceClass, props.InstanceSize),
		MachineImage:         createMachineImage(),
		SecurityGroup: createAsgSecurityGroup(scope, jsii.String(props.Name+"SecurityGroup"), &securityGroupProps{
			Name:        props.Name + "SecurityGroup",
			Description: "SecurityGroup for " + props.Name,
			vpc:         vpc,
		}),

		UserData:   ec2.UserData_ForLinux(&ec2.LinuxUserDataOptions{Shebang: jsii.String("#!/bin/bash")}),
		VpcSubnets: &ec2.SubnetSelection{SubnetType: ec2.SubnetType_PUBLIC},
		Vpc:        vpc,
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

func createMachineImage() ec2.IMachineImage {
	image := ec2.NewAmazonLinuxImage(&ec2.AmazonLinuxImageProps{
		CpuType:        ec2.AmazonLinuxCpuType_X86_64,
		Edition:        ec2.AmazonLinuxEdition_STANDARD,
		Generation:     ec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		Virtualization: ec2.AmazonLinuxVirt_HVM,
		Kernel:         ec2.AmazonLinuxKernel_KERNEL5_X,
	})
	return image
}

func createCapacityProvider(scope constructs.Construct, id *string, props *ContainerComputeAsgCapacityProviderProps, asg autoscaling.IAutoScalingGroup) ecs.AsgCapacityProvider {
	asgCapacityProvider := ecs.NewAsgCapacityProvider(scope, id, &ecs.AsgCapacityProviderProps{
		AutoScalingGroup:                   asg,
		EnableManagedScaling:               jsii.Bool(true),
		EnableManagedTerminationProtection: jsii.Bool(false),
		TargetCapacityPercent:              jsii.Number(100),
		CapacityProviderName:               jsii.String(props.Name),
		CanContainersAccessInstanceRole:    jsii.Bool(true),
	})
	return asgCapacityProvider
}
