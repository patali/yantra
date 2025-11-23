# Yantra Documentation

Welcome to the Yantra documentation! This directory contains comprehensive guides for understanding, developing, and deploying Yantra.

## Getting Started

**New to Yantra?** Start here:

1. [Getting Started Guide](./GETTING_STARTED.md) - Quick setup and installation
2. [Configuration Guide](./CONFIGURATION.md) - Environment variables and settings

## Reference Documentation

### For Users

- **[Node Types](./NODE_TYPES.md)** - Complete reference for all available workflow nodes
- **[API Reference](./API.md)** - REST API endpoints and examples

### For Developers

- **[Architecture](./ARCHITECTURE.md)** - System design, principles, and technology stack
- **[Backend Architecture](../backend/docs/WORKFLOW_ARCHITECTURE.md)** - Detailed workflow engine design
- **[Outbox Pattern](../backend/docs/OUTBOX_ARCHITECTURE.md)** - Reliable side-effect execution

### For DevOps

- **[Deployment Guide](./DEPLOYMENT.md)** - Production deployment instructions
- **[Configuration Guide](./CONFIGURATION.md)** - Environment and security setup

## Documentation Structure

```
docs/
├── README.md               # This file
├── GETTING_STARTED.md      # Quick start guide
├── CONFIGURATION.md        # Environment configuration
├── API.md                  # REST API reference
├── NODE_TYPES.md           # Workflow nodes reference
├── ARCHITECTURE.md         # System architecture
└── DEPLOYMENT.md           # Production deployment

backend/docs/
├── WORKFLOW_ARCHITECTURE.md  # Workflow engine details
└── OUTBOX_ARCHITECTURE.md    # Outbox pattern details
```

## Quick Links

### Common Tasks

- **Setup for Development**: [Getting Started - Without Docker](./GETTING_STARTED.md#local-development-without-docker)
- **Run with Docker**: [Getting Started - With Docker](./GETTING_STARTED.md#quick-start-with-docker)
- **Configure Email**: [Configuration - Email Node Configuration](./CONFIGURATION.md#email-node-configuration)
- **Deploy to Production**: [Deployment Guide](./DEPLOYMENT.md)
- **Add New Node Type**: [Architecture - Node Executor Pattern](./ARCHITECTURE.md#node-executor-pattern)

### API Examples

- **Execute Workflow**: [API - Execute Workflow](./API.md#execute-workflow)
- **Create Workflow**: [API - Create Workflow](./API.md#create-workflow)
- **Schedule Workflow**: [API - Update Workflow Schedule](./API.md#update-workflow-schedule)
- **Trigger via Webhook**: [API - Trigger Workflow via Webhook](./API.md#trigger-workflow-via-webhook)

### Node Examples

- **Conditional Branching**: [Node Types - Conditional Node](./NODE_TYPES.md#conditional-node)
- **HTTP API Calls**: [Node Types - HTTP Node](./NODE_TYPES.md#http-node)
- **Loop Processing**: [Node Types - Loop Node](./NODE_TYPES.md#loop-node)
- **Long Delays**: [Node Types - Sleep Node](./NODE_TYPES.md#sleep-node)

## Contributing to Documentation

When adding or updating documentation:

1. Keep explanations clear and concise
2. Include code examples where helpful
3. Add links to related documentation
4. Update this README if adding new docs
5. Follow the existing format and style

## Need Help?

- **Bug Reports**: Open an issue on GitHub
- **Feature Requests**: Open an issue with `[Feature Request]` prefix
- **Questions**: Check existing documentation first, then ask in issues
- **Security Issues**: Email security@example.com

## Documentation Versions

This documentation corresponds to Yantra v1.x. For older versions, see the git history or archived docs.

---

**Happy automating! 🚀**

