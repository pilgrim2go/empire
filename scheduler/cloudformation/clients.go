package cloudformation

import (
	"time"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pmylund/go-cache"
)

// cacher duck types the go-cache interface.
type cacher interface {
	Set(k string, x interface{}, d time.Duration)
	Get(k string) (interface{}, bool)
}

// cachingECSClient wraps an ecsClient to perform some performance
// optimizations, by taking advantage of the fact that task definitions are
// essentially immutable and can be cached forever.
type cachingECSClient struct {
	ecsClient

	// cache of task definitions
	taskDefinitions cacher
}

// ecsWithCaching wraps an ecs.ECS client with caching.
func ecsWithCaching(ecs *ecs.ECS) *cachingECSClient {
	return &cachingECSClient{
		ecsClient:       ecs,
		taskDefinitions: cache.New(30*time.Minute, 30*time.Second),
	}
}

// DescribeTaskDefinition will use the task definition from cache if provided
// with a task definition ARN.
func (c *cachingECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	// TODO: Check if the task definition is an arn.
	if v, ok := c.taskDefinitions.Get(*input.TaskDefinition); ok {
		return &ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: v.(*ecs.TaskDefinition),
		}, nil
	}

	resp, err := c.ecsClient.DescribeTaskDefinition(input)
	if err != nil {
		return resp, err
	}

	c.taskDefinitions.Set(*resp.TaskDefinition.TaskDefinitionArn, resp.TaskDefinition, 0)

	return resp, err
}
