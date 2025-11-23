#!/bin/sh

set -e

echo "🔄 Starting Yantra Server..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL is not set"
    exit 1
fi

echo "🗄️  Database URL configured"
echo "📦 Migrations will run automatically on startup"

# Start the application (migrations run automatically inside the app)
echo "🚀 Starting Yantra server..."
exec ./yantra-server
