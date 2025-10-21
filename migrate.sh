#!/bin/bash

set -e  # Exit on any error

echo "🔄 Starting migration process..."

# Load environment variables from .env file
if [ -f .env ]; then
    echo "📝 Loading environment variables from .env..."
    export $(cat .env | grep -v '^#' | grep -v '^[[:space:]]*$' | xargs)
else
    echo "⚠️  Warning: .env file not found. Using environment variables."
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL is not set. Please set it in .env or environment."
    exit 1
fi

echo "🗄️  Database URL: ${DATABASE_URL}"

# Step 1: Install River CLI if not already installed
echo ""
echo "📦 Installing River CLI..."
if command -v river &> /dev/null; then
    echo "✅ River CLI already installed"
else
    echo "⬇️  Installing river..."
    go install github.com/riverqueue/river/cmd/river@latest
fi

# Step 2: Run River migrations
echo ""
echo "🌊 Running River migrations..."
river migrate-up --database-url "$DATABASE_URL"
echo "✅ River migrations completed"

# Step 3: Build the GORM migration binary
echo ""
echo "🔨 Building GORM migration binary..."
go build -o ./bin/migrate ./cmd/migrate/main.go
echo "✅ Migration binary built at ./bin/migrate"

# Step 4: Run GORM migrations
echo ""
echo "🗃️  Running GORM migrations..."
./bin/migrate
echo "✅ GORM migrations completed"

echo ""
echo "🎉 All migrations completed successfully!"
