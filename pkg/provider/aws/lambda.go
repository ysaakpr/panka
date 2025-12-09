package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// LambdaProvider implements Lambda function management
type LambdaProvider struct {
	provider *Provider
	client   *lambda.Client
}

// NewLambdaProvider creates a new Lambda provider
func NewLambdaProvider(p *Provider) *LambdaProvider {
	return &LambdaProvider{
		provider: p,
		client:   lambda.NewFromConfig(p.GetConfig()),
	}
}

// Create creates a new Lambda function
func (lp *LambdaProvider) Create(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	lambdaResource, ok := resource.(*schema.Lambda)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "create",
			Message:   "invalid resource type for Lambda provider",
		}
	}

	functionName := fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		lambdaResource.Metadata.Name,
	)

	lp.provider.GetLogger().Info("Creating Lambda function",
		zap.String("name", functionName),
		zap.String("runtime", lambdaResource.Spec.Runtime),
	)

	if opts.DryRun {
		return &provider.ResourceResult{
			ResourceID: functionName,
			Kind:       schema.KindLambda,
			Status:     provider.StatusPending,
			Outputs: map[string]string{
				"function_name": functionName,
				"runtime":       lambdaResource.Spec.Runtime,
			},
			Timestamp: time.Now(),
		}, nil
	}

	// Check if IAM role is specified
	if lambdaResource.Spec.RoleArn == "" {
		lp.provider.GetLogger().Warn("Lambda function requires IAM role - returning placeholder",
			zap.String("function", functionName),
		)
		// Return a placeholder result - Lambda requires an IAM execution role
		// In a full implementation, we would auto-create the role
		return &provider.ResourceResult{
			ResourceID: functionName,
			Kind:       schema.KindLambda,
			Status:     provider.StatusPending,
			Outputs: map[string]string{
				"function_name": functionName,
				"runtime":       lambdaResource.Spec.Runtime,
				"status":        "role_required",
				"message":       "Lambda function requires an IAM execution role. Specify 'roleArn' in the spec or use 'panka admin' to configure auto-provisioning.",
			},
			Timestamp: time.Now(),
		}, nil
	}

	// Check if code source is specified
	if lambdaResource.Spec.Code.ImageUri == "" && lambdaResource.Spec.Code.S3Bucket == "" {
		lp.provider.GetLogger().Warn("Lambda function requires code - returning placeholder",
			zap.String("function", functionName),
		)
		return &provider.ResourceResult{
			ResourceID: functionName,
			Kind:       schema.KindLambda,
			Status:     provider.StatusPending,
			Outputs: map[string]string{
				"function_name": functionName,
				"runtime":       lambdaResource.Spec.Runtime,
				"status":        "code_required",
				"message":       "Lambda function requires code. Specify 's3Bucket/s3Key' or 'imageUri' in the spec.",
			},
			Timestamp: time.Now(),
		}, nil
	}

	// Build tags
	tags := lp.provider.tagHelper.BuildTags(opts, resource)

	// Parse memory (default 128MB)
	memory := int32(128)
	if lambdaResource.Spec.Memory != "" {
		if m, err := strconv.Atoi(lambdaResource.Spec.Memory); err == nil {
			memory = int32(m)
		}
	}

	// Parse timeout (default 30 seconds)
	timeout := int32(30)
	if lambdaResource.Spec.Timeout != "" {
		if t, err := strconv.Atoi(lambdaResource.Spec.Timeout); err == nil {
			timeout = int32(t)
		}
	}

	// Build environment variables
	var envVars map[string]string
	if len(lambdaResource.Spec.Environment) > 0 {
		envVars = make(map[string]string)
		for k, v := range lambdaResource.Spec.Environment {
			envVars[k] = fmt.Sprintf("%v", v)
		}
	}

	// Determine code source (already validated above)
	var code *types.FunctionCode
	if lambdaResource.Spec.Code.ImageUri != "" {
		code = &types.FunctionCode{
			ImageUri: aws.String(lambdaResource.Spec.Code.ImageUri),
		}
	} else {
		code = &types.FunctionCode{
			S3Bucket: aws.String(lambdaResource.Spec.Code.S3Bucket),
			S3Key:    aws.String(lambdaResource.Spec.Code.S3Key),
		}
	}

	// Create function input
	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Runtime:      types.Runtime(lambdaResource.Spec.Runtime),
		Handler:      aws.String(lambdaResource.Spec.Handler),
		Role:         aws.String(lambdaResource.Spec.RoleArn), // Note: Role must exist
		Code:         code,
		MemorySize:   aws.Int32(memory),
		Timeout:      aws.Int32(timeout),
		Tags:         tags,
	}

	// Add environment if specified
	if len(envVars) > 0 {
		input.Environment = &types.Environment{
			Variables: envVars,
		}
	}

	// Add VPC config if enabled
	if lambdaResource.Spec.VPC.Enabled {
		if len(lambdaResource.Spec.VPC.SubnetIds) > 0 || len(lambdaResource.Spec.VPC.SecurityGroupIds) > 0 {
			input.VpcConfig = &types.VpcConfig{
				SubnetIds:        lambdaResource.Spec.VPC.SubnetIds,
				SecurityGroupIds: lambdaResource.Spec.VPC.SecurityGroupIds,
			}
		}
	}

	// Add layers if specified
	if len(lambdaResource.Spec.Layers) > 0 {
		input.Layers = lambdaResource.Spec.Layers
	}

	// Create function
	result, err := lp.client.CreateFunction(ctx, input)
	if err != nil {
		lp.provider.GetLogger().Error("Failed to create Lambda function",
			zap.String("function", functionName),
			zap.Error(err),
		)
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "create",
			ResourceID: functionName,
			Cause:      err,
			Message:    "failed to create Lambda function",
		}
	}

	lp.provider.GetLogger().Info("Lambda function created",
		zap.String("function", functionName),
		zap.String("arn", *result.FunctionArn),
	)

	return &provider.ResourceResult{
		ResourceID: functionName,
		Kind:       schema.KindLambda,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"function_name": functionName,
			"function_arn":  *result.FunctionArn,
			"runtime":       string(result.Runtime),
			"handler":       *result.Handler,
			"memory_mb":     fmt.Sprintf("%d", *result.MemorySize),
			"timeout_sec":   fmt.Sprintf("%d", *result.Timeout),
		},
		Timestamp: time.Now(),
	}, nil
}

// Read reads the current state of a Lambda function
func (lp *LambdaProvider) Read(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	result, err := lp.client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(resourceID),
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "read",
			ResourceID: resourceID,
			Cause:      err,
			Message:    "failed to get Lambda function",
		}
	}

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindLambda,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"function_name": *result.Configuration.FunctionName,
			"function_arn":  *result.Configuration.FunctionArn,
			"runtime":       string(result.Configuration.Runtime),
			"handler":       *result.Configuration.Handler,
			"state":         string(result.Configuration.State),
		},
		Timestamp: time.Now(),
	}, nil
}

// Update updates an existing Lambda function
func (lp *LambdaProvider) Update(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	lambdaResource, ok := resource.(*schema.Lambda)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "update",
			Message:   "invalid resource type for Lambda provider",
		}
	}

	functionName := fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		lambdaResource.Metadata.Name,
	)

	lp.provider.GetLogger().Info("Updating Lambda function",
		zap.String("function", functionName),
	)

	// Parse memory and timeout
	memory := int32(128)
	if lambdaResource.Spec.Memory != "" {
		if m, err := strconv.Atoi(lambdaResource.Spec.Memory); err == nil {
			memory = int32(m)
		}
	}
	timeout := int32(30)
	if lambdaResource.Spec.Timeout != "" {
		if t, err := strconv.Atoi(lambdaResource.Spec.Timeout); err == nil {
			timeout = int32(t)
		}
	}

	// Build environment variables
	var envVars map[string]string
	if len(lambdaResource.Spec.Environment) > 0 {
		envVars = make(map[string]string)
		for k, v := range lambdaResource.Spec.Environment {
			envVars[k] = fmt.Sprintf("%v", v)
		}
	}

	// Update configuration
	input := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(functionName),
		Handler:      aws.String(lambdaResource.Spec.Handler),
		MemorySize:   aws.Int32(memory),
		Timeout:      aws.Int32(timeout),
	}

	if len(envVars) > 0 {
		input.Environment = &types.Environment{
			Variables: envVars,
		}
	}

	result, err := lp.client.UpdateFunctionConfiguration(ctx, input)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "update",
			ResourceID: functionName,
			Cause:      err,
			Message:    "failed to update Lambda function",
		}
	}

	return &provider.ResourceResult{
		ResourceID: functionName,
		Kind:       schema.KindLambda,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"function_name": *result.FunctionName,
			"function_arn":  *result.FunctionArn,
		},
		Timestamp: time.Now(),
	}, nil
}

// Delete deletes a Lambda function
func (lp *LambdaProvider) Delete(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	lp.provider.GetLogger().Info("Deleting Lambda function",
		zap.String("function", resourceID),
	)

	_, err := lp.client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(resourceID),
	})
	if err != nil {
		// Check if the function doesn't exist - that's okay, treat as success
		errStr := err.Error()
		if strings.Contains(errStr, "ResourceNotFoundException") || strings.Contains(errStr, "Function not found") {
			lp.provider.GetLogger().Info("Lambda function already deleted or never existed",
				zap.String("function", resourceID),
			)
			return &provider.ResourceResult{
				ResourceID: resourceID,
				Kind:       schema.KindLambda,
				Status:     provider.StatusDeleted,
				Timestamp:  time.Now(),
			}, nil
		}

		lp.provider.GetLogger().Error("Failed to delete Lambda function",
			zap.String("function", resourceID),
			zap.Error(err),
		)
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "delete",
			ResourceID: resourceID,
			Cause:      err,
			Message:    "failed to delete Lambda function",
		}
	}

	lp.provider.GetLogger().Info("Lambda function deleted",
		zap.String("function", resourceID),
	)

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindLambda,
		Status:     provider.StatusDeleted,
		Timestamp:  time.Now(),
	}, nil
}

// Exists checks if a Lambda function exists
func (lp *LambdaProvider) Exists(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (bool, error) {
	_, err := lp.client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(resourceID),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}

// GetOutputs returns the outputs of a Lambda function
func (lp *LambdaProvider) GetOutputs(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (map[string]string, error) {
	result, err := lp.Read(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return result.Outputs, nil
}

