package main

import (
	clusterConstruct "cdk-consrtuct/compute-construct"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkConsrtuctStackProps struct {
	awscdk.StackProps
}

func ComputeStack(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	vpcId := "vpc-535bd136"
	stack := awscdk.NewStack(scope, &id, &sprops)

	clusterConstruct.NewContainerCompute(stack, jsii.String("DevComputeStack"), &clusterConstruct.ContainerComputeProps{
		Vpc: clusterConstruct.VpcProps{VpcId: vpcId},
		Cluster: clusterConstruct.ContainerComputeClusterProps{
			ClusterName:       "DemoGoCluster",
			ContainerInsights: false,
		},
		SecurityGroup: clusterConstruct.ContainerComputeSecurityGroupProps{
			AsgSgName:                 jsii.String("MicroAsgSg"),
			AsgSgDescription:          jsii.String("Security group for Asg"),
			LoadbalancerSgName:        jsii.String("LbSg"),
			LoadbalancerSgDescription: jsii.String("Security group for Load balancer"),
		},
		AsgCapacityProviderProps: []*clusterConstruct.ContainerComputeAsgCapacityProviderProps{
			{
				AutoScalingGroupName: jsii.String("MicroAsg"),
				InstanceClass:        awsec2.InstanceClass_BURSTABLE2,
				InstanceSize:         awsec2.InstanceSize_MICRO,
				MinCapacity:          jsii.Number(0),
				MaxCapacity:          jsii.Number(1),
				SshKeyName:           jsii.String("breezethru-demo-key-pair"),
				CapacityProviderName: jsii.String("MicroAsgCapacityProvider"),
			},
			{
				AutoScalingGroupName: jsii.String("SmallAsg"),
				InstanceClass:        awsec2.InstanceClass_BURSTABLE2,
				InstanceSize:         awsec2.InstanceSize_SMALL,
				MinCapacity:          jsii.Number(0),
				MaxCapacity:          jsii.Number(1),
				SshKeyName:           jsii.String("breezethru-demo-key-pair"),
				CapacityProviderName: jsii.String("SmallAsgCapacityProvider"),
			},
		},
		LoadBalancerProps: &clusterConstruct.ContainerComputeLoadBalancerProps{
			LoadBalancerName:       jsii.String("DemoLoadBalancerGoLang"),
			ListenerCertificateArn: jsii.String("arn:aws:acm:us-east-1:305251478828:certificate/3f5f3c4f-5e6c-40de-a588-41cca514bbeb"),
		},
		CloudmapNamespaceProps: &clusterConstruct.ContainerComputeCloudmapNamespaceProps{
			Name:        jsii.String("DemoServiceNamespace"),
			Description: jsii.String("Demo namespace for golang"),
		},
	})
	return stack
}

func demo_service(scope constructs.Construct, id string, props *CdkConsrtuctStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	vpcId := "vpc-535bd136"
	vpcName := "lookupvpc"
	vpc := awsec2.Vpc_FromLookup(stack, &vpcName, &awsec2.VpcLookupOptions{VpcId: &vpcId})
	sg := awsec2.SecurityGroup_FromLookupById(stack, jsii.String("SgLookup"), jsii.String("sg-0d69523be628ee1bf"))

	taskdefinition := awsecs.NewTaskDefinition(stack, jsii.String("DemoTaskDef"), &awsecs.TaskDefinitionProps{
		Family:        jsii.String("DemoTaskDefinition"),
		NetworkMode:   awsecs.NetworkMode_AWS_VPC,
		Compatibility: awsecs.Compatibility_EC2,
	})

	containerDefinition := awsecs.NewContainerDefinition(stack, jsii.String("ContainerDefinition"), &awsecs.ContainerDefinitionProps{
		Image:         awsecs.ContainerImage_FromRegistry(jsii.String("nginx"), &awsecs.RepositoryImageProps{}),
		ContainerName: jsii.String("NginxDemo"),
		Essential:     jsii.Bool(true),
		PortMappings: &[]*awsecs.PortMapping{{
			ContainerPort: jsii.Number(80),
			Protocol:      awsecs.Protocol_TCP,
		},
			{
				ContainerPort: jsii.Number(443),
				Protocol:      awsecs.Protocol_TCP,
			}},
		Cpu:            jsii.Number(512),
		MemoryLimitMiB: jsii.Number(950),
		Logging: awsecs.AwsLogDriver_AwsLogs(&awsecs.AwsLogDriverProps{
			LogGroup: awslogs.NewLogGroup(stack, jsii.String("DemoLogGroup"), &awslogs.LogGroupProps{
				LogGroupName:  jsii.String("DemoEcsLogGroup"),
				RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
				Retention:     awslogs.RetentionDays_ONE_DAY,
			}),
			StreamPrefix: jsii.String("/ecs/demo"),
		}),
		TaskDefinition: taskdefinition,
	})

	service := awsecs.NewEc2Service(stack, jsii.String("EcsService"), &awsecs.Ec2ServiceProps{
		Cluster: awsecs.Cluster_FromClusterAttributes(stack, jsii.String("LookupCluster"), &awsecs.ClusterAttributes{
			ClusterName: jsii.String("DemoGoCluster"),
			Vpc:         vpc,
			SecurityGroups: &[]awsec2.ISecurityGroup{
				sg,
			},
		}),
		CircuitBreaker: &awsecs.DeploymentCircuitBreaker{Rollback: jsii.Bool(true)},
		TaskDefinition: taskdefinition,
		DesiredCount:   jsii.Number(2),
		ServiceName:    jsii.String("DemoEcsService"),
		CapacityProviderStrategies: &[]*awsecs.CapacityProviderStrategy{{
			CapacityProvider: jsii.String("SmallAsgCapacityProvider"),
			Weight:           jsii.Number(1),
		}},
		// CloudMapOptions: &awsecs.CloudMapOptions{
		// 	CloudMapNamespace: awsservicediscovery.PrivateDnsNamespace_FromPrivateDnsNamespaceAttributes(stack, jsii.String("LookUpDns"), &awsservicediscovery.PrivateDnsNamespaceAttributes{
		// 		NamespaceArn:  jsii.String("arn:aws:servicediscovery:us-east-1:305251478828:namespace/ns-5n44ig2wzxe6offa"),
		// 		NamespaceId:   jsii.String("ns-5n44ig2wzxe6offa"),
		// 		NamespaceName: jsii.String("DemoServiceNamespace"),
		// 	}),
		// 	DnsRecordType: awsservicediscovery.DnsRecordType_A,
		// 	ContainerPort: jsii.Number(80),
		// 	DnsTtl:        awscdk.Duration_Seconds(jsii.Number(60)),
		// },
	})

	tg := awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String("TargetGroup"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
		TargetGroupName: jsii.String("NginxTg"),
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			Enabled:          jsii.Bool(true),
			HealthyHttpCodes: jsii.String("200"),
			Path:             jsii.String("/"),
			Interval:         awscdk.Duration_Seconds(jsii.Number(30)),
		},
		TargetType: awselasticloadbalancingv2.TargetType_IP,
		Vpc:        vpc,
		Protocol:   awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		Targets: &[]awselasticloadbalancingv2.IApplicationLoadBalancerTarget{
			service.LoadBalancerTarget(&awsecs.LoadBalancerTargetOptions{
				ContainerName: containerDefinition.ContainerName(),
				ContainerPort: jsii.Number(80),
				Protocol:      awsecs.Protocol_TCP,
			}),
		},
	})

	awselasticloadbalancingv2.NewApplicationListenerRule(stack, jsii.String("ListenerRule"), &awselasticloadbalancingv2.ApplicationListenerRuleProps{
		Priority: jsii.Number(2),
		Action:   awselasticloadbalancingv2.ListenerAction_Forward(&[]awselasticloadbalancingv2.IApplicationTargetGroup{tg}, &awselasticloadbalancingv2.ForwardOptions{}),
		Conditions: &[]awselasticloadbalancingv2.ListenerCondition{
			awselasticloadbalancingv2.ListenerCondition_HostHeaders(jsii.Strings("nginx.dynamostack.com")),
			awselasticloadbalancingv2.ListenerCondition_PathPatterns(jsii.Strings("/*")),
		},
		Listener: awselasticloadbalancingv2.ApplicationListener_FromApplicationListenerAttributes(stack, jsii.String("LookUpListener"), &awselasticloadbalancingv2.ApplicationListenerAttributes{
			ListenerArn:   jsii.String("arn:aws:elasticloadbalancing:us-east-1:305251478828:listener/app/DemoLoadBalancerGoLang/964c9cf323a5a5bc/6fbdb3e4127c8606"),
			SecurityGroup: awsec2.SecurityGroup_FromLookupById(stack, jsii.String("LbSg"), jsii.String("sg-0860f3bf2591663b2")),
		}),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	demo_service(app, "DemoService", &CdkConsrtuctStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	ComputeStack(app, "ComputeStack", &CdkConsrtuctStackProps{
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
