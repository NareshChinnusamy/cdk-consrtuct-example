package containerenvironment

import (
	props "infra/ci-cd/props"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkConsrtuctStackProps struct {
	awscdk.StackProps
}

func CreateTaskDefinition(scope constructs.Construct, id string, props *props.TeqChargingIacStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	taskdefinition := awsecs.NewTaskDefinition(stack, jsii.String("DemoTaskDef"), &awsecs.TaskDefinitionProps{
		Family:        jsii.String("DbHealthCheckTaskDefinition"),
		NetworkMode:   awsecs.NetworkMode_AWS_VPC,
		Compatibility: awsecs.Compatibility_FARGATE,
		Cpu:           jsii.String("1024"),
		MemoryMiB:     jsii.String("2048"),
	})

	awsecs.NewContainerDefinition(stack, jsii.String("ContainerDefinition"), &awsecs.ContainerDefinitionProps{
		Image:         awsecs.AssetImage_FromRegistry(jsii.String("mariadb:10.7"), &awsecs.RepositoryImageProps{}),
		ContainerName: jsii.String("MariaDB"),
		Essential:     jsii.Bool(true),
		PortMappings: &[]*awsecs.PortMapping{{
			ContainerPort: jsii.Number(3306),
			Protocol:      awsecs.Protocol_TCP,
		},
		},
		Logging: awsecs.AwsLogDriver_AwsLogs(&awsecs.AwsLogDriverProps{
			LogGroup: awslogs.NewLogGroup(stack, jsii.String("DemoLogGroup"), &awslogs.LogGroupProps{
				LogGroupName:  jsii.String("DemoEcsLogGroup"),
				RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
				Retention:     awslogs.RetentionDays_ONE_DAY,
			}),
			StreamPrefix: jsii.String("/ecs/demo"),
		}),
		TaskDefinition: taskdefinition,
		HealthCheck: &awsecs.HealthCheck{
			Command:  jsii.Strings("CMD-SHELL", "curl http://localhost:8080/health-check || exit 1"),
			Interval: awscdk.Duration_Seconds(jsii.Number(5)),
			Retries:  jsii.Number(3),
		},
		Environment: &map[string]*string{
			"MYSQL_ROOT_PASSWORD": jsii.String("password"),
		},
	})
	return stack
}
