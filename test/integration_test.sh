#!/bin/bash
# Integration test runner for Panka AWS providers

set -e

echo "🚀 Starting LocalStack..."
docker-compose -f test/docker-compose.localstack.yml up -d

echo "⏳ Waiting for LocalStack to be ready..."
sleep 5

# Check if LocalStack is ready
echo "🔍 Checking LocalStack health..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -s http://localhost:4566/_localstack/health | grep -q '"s3"'; then
        echo "✅ LocalStack is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "   Attempt $attempt/$max_attempts..."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ LocalStack failed to start"
    docker-compose -f test/docker-compose.localstack.yml logs
    exit 1
fi

echo ""
echo "🧪 Running integration tests..."
export LOCALSTACK_ENDPOINT=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

go test -tags=integration -v ./pkg/provider/aws/... -count=1

echo ""
echo "🛑 Stopping LocalStack..."
docker-compose -f test/docker-compose.localstack.yml down

echo "✅ Integration tests complete!"

