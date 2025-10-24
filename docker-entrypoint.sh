#!/bin/sh

set -e

echo "🔄 Starting Yantra Server..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL is not set"
    exit 1
fi

echo "🗄️  Database URL configured"

# Run River migrations
echo "🌊 Running River migrations..."
if river migrate-up --database-url "$DATABASE_URL"; then
    echo "✅ River migrations completed"
else
    echo "⚠️  River migrations failed, but continuing startup..."
fi

# Start the application
echo "🚀 Starting Yantra server..."
exec ./yantra-server
