package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// S3Backend implements the Backend interface using AWS S3
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
	logger *zap.Logger
}

// S3BackendConfig holds S3-specific configuration
type S3BackendConfig struct {
	Client *s3.Client
	Bucket string
	Prefix string
	Logger *zap.Logger
}

// NewS3Backend creates a new S3 state backend
func NewS3Backend(cfg *S3BackendConfig) (*S3Backend, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("S3 client is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Use a no-op logger if none provided
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &S3Backend{
		client: cfg.Client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
		logger: logger,
	}, nil
}

// Save saves the state to S3
func (b *S3Backend) Save(ctx context.Context, key string, state *State) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Build full S3 key
	s3Key := b.buildKey(key)

	// Update state metadata
	state.LastUpdate = time.Now()
	state.Metadata.UpdatedAt = state.LastUpdate

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Upload to S3
	b.logger.Info("Saving state to S3",
		zap.String("bucket", b.bucket),
		zap.String("key", s3Key),
		zap.Int("size", len(data)),
	)

	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
		Metadata: map[string]string{
			"stack":       state.Metadata.Stack,
			"environment": state.Metadata.Environment,
			"version":     state.Version,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload state to S3: %w", err)
	}

	b.logger.Info("State saved successfully",
		zap.String("key", s3Key),
		zap.Int("resources", len(state.Resources)),
	)

	return nil
}

// Load loads the state from S3
func (b *S3Backend) Load(ctx context.Context, key string) (*State, error) {
	s3Key := b.buildKey(key)

	b.logger.Debug("Loading state from S3",
		zap.String("bucket", b.bucket),
		zap.String("key", s3Key),
	)

	// Get object from S3
	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(s3Key),
	})

	if err != nil {
		// Check if object doesn't exist
		if strings.Contains(err.Error(), "NoSuchKey") {
			return nil, fmt.Errorf("state not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get state from S3: %w", err)
	}
	defer result.Body.Close()

	// Read response body
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	// Unmarshal state
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	b.logger.Info("State loaded successfully",
		zap.String("key", s3Key),
		zap.String("stack", state.Metadata.Stack),
		zap.Int("resources", len(state.Resources)),
	)

	return &state, nil
}

// Exists checks if a state exists in S3
func (b *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	s3Key := b.buildKey(key)

	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(s3Key),
	})

	if err != nil {
		// Check if object doesn't exist
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if state exists: %w", err)
	}

	return true, nil
}

// Delete deletes the state from S3
func (b *S3Backend) Delete(ctx context.Context, key string) error {
	s3Key := b.buildKey(key)

	b.logger.Info("Deleting state from S3",
		zap.String("bucket", b.bucket),
		zap.String("key", s3Key),
	)

	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(s3Key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete state from S3: %w", err)
	}

	b.logger.Info("State deleted successfully", zap.String("key", s3Key))
	return nil
}

// List lists all state keys with the given prefix
func (b *S3Backend) List(ctx context.Context, prefix string) ([]string, error) {
	searchPrefix := b.buildKey(prefix)

	b.logger.Debug("Listing states in S3",
		zap.String("bucket", b.bucket),
		zap.String("prefix", searchPrefix),
	)

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(searchPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				// Remove the prefix to get relative key
				relKey := strings.TrimPrefix(*obj.Key, b.prefix)
				relKey = strings.TrimPrefix(relKey, "/")
				keys = append(keys, relKey)
			}
		}
	}

	b.logger.Debug("Listed states", zap.Int("count", len(keys)))
	return keys, nil
}

// ListVersions lists all versions of a state
func (b *S3Backend) ListVersions(ctx context.Context, key string) ([]*StateVersion, error) {
	s3Key := b.buildKey(key)

	b.logger.Debug("Listing state versions",
		zap.String("bucket", b.bucket),
		zap.String("key", s3Key),
	)

	result, err := b.client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(s3Key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	var versions []*StateVersion
	for _, ver := range result.Versions {
		if ver.VersionId == nil || ver.Key == nil {
			continue
		}

		version := &StateVersion{
			VersionID:  *ver.VersionId,
			Size:       *ver.Size,
			ModifiedAt: *ver.LastModified,
			IsLatest:   ver.IsLatest != nil && *ver.IsLatest,
		}
		versions = append(versions, version)
	}

	// Sort by modified time (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ModifiedAt.After(versions[j].ModifiedAt)
	})

	b.logger.Debug("Listed versions", zap.Int("count", len(versions)))
	return versions, nil
}

// GetVersion gets a specific version of the state
func (b *S3Backend) GetVersion(ctx context.Context, key string, versionID string) (*State, error) {
	s3Key := b.buildKey(key)

	b.logger.Debug("Getting state version",
		zap.String("bucket", b.bucket),
		zap.String("key", s3Key),
		zap.String("version", versionID),
	)

	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:    aws.String(b.bucket),
		Key:       aws.String(s3Key),
		VersionId: aws.String(versionID),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get state version: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Close closes the S3 backend (no-op for S3)
func (b *S3Backend) Close() error {
	b.logger.Info("Closing S3 backend")
	return nil
}

// buildKey builds the full S3 key with prefix
func (b *S3Backend) buildKey(key string) string {
	if b.prefix == "" {
		return key
	}
	return filepath.Join(b.prefix, key)
}

// Ensure S3Backend implements Backend interface
var _ Backend = (*S3Backend)(nil)

