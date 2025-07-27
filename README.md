# 🚀 Kubernetes Orchestrator

A beautiful, terminal-based Kubernetes cluster management tool with GitOps integration, built with Go and Bubble Tea TUI.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.19+-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## ✨ Features

### 🎯 **Multi-Cluster Management**
- **Interactive Cluster Selection**: Beautiful TUI for switching between clusters
- **Dynamic Cluster Addition**: Add new clusters with step-by-step wizard
- **Connection Validation**: Automatic verification of cluster connectivity
- **Persistent Configuration**: Clusters persist across sessions

### 🖥️ **Beautiful Terminal Interface**
- **Modern TUI**: Built with Bubble Tea for responsive, beautiful interface
- **Real-time Command Execution**: Execute kubectl commands directly in terminal
- **Syntax Highlighting**: Color-coded output and status indicators
- **Loading States**: Professional loading animations and progress indicators

### 🚀 **GitOps Integration**
- **Automatic Git Sync**: Resource modifications auto-sync to Git repository
- **ArgoCD Ready**: Seamless integration with ArgoCD workflows
- **Resource Export**: Export cluster resources to YAML files
- **Commit Tracking**: Automatic commits with timestamps and descriptions

### 🔧 **System Safety**
- **Dependency Validation**: Verifies kubectl and git availability on startup
- **Connection Testing**: Tests cluster connectivity before operations
- **Error Handling**: Comprehensive error handling with helpful messages
- **Safe Operations**: Validates operations before execution

## 🏗️ Architecture

The application is organized into clean, modular packages:

```
kube-orchestrator/
├── main.go                    # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   │   └── manager.go        # Cluster registry and kubeconfig handling
│   ├── system/               # System dependency management
│   │   └── dependencies.go   # kubectl and git validation
│   ├── kubectl/              # Kubernetes operations
│   │   └── executor.go       # kubectl command execution
│   ├── git/                  # Git operations
│   │   └── manager.go        # GitOps workflow management
│   └── ui/                   # User interface
│       ├── application.go    # Main TUI application
│       ├── styles.go         # UI styling definitions
│       ├── views.go          # View rendering logic
│       └── items.go          # List item definitions
├── Makefile                  # Build system
└── README.md                 # This file
```

## 📋 Prerequisites

### Required Dependencies
- **kubectl** - Kubernetes command-line tool
  - Installation: https://kubernetes.io/docs/tasks/tools/
- **git** - Version control system
  - Installation: https://git-scm.com/downloads

### Optional Dependencies
- **Go 1.19+** - For building from source
- **Docker** - For containerized builds

## 🚀 Quick Start

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/your-org/kube-orchestrator.git
cd kube-orchestrator

# Install dependencies and build
make setup-dev
make build

# Run the application
./build/kube-orchestrator
```

### Option 2: Using Make (Recommended)

```bash
# Setup development environment (installs tools and dependencies)
make setup-dev

# Build and run in development mode
make run

# Or build for production
make build && ./build/kube-orchestrator
```

### Option 3: Install Globally

```bash
make build
make install  # Installs to /usr/local/bin
kube-orchestrator
```

## 🎮 Usage

### First Run
1. **Dependency Check**: The application automatically verifies kubectl and git
2. **Cluster Selection**: Choose from existing clusters or add a new one
3. **Add New Cluster**: Follow the step-by-step wizard to add clusters

### Adding a New Cluster
1. Select "➕ Add New Cluster" from the main menu
2. Enter cluster name (e.g., `production`, `staging`)
3. Enter public IP or DNS (e.g., `prod.k8s.company.com`)
4. Provide path to kubeconfig file with admin permissions
5. The system will verify connectivity and add the cluster

### Terminal Commands

#### Built-in Commands
```bash
help              # Show help information
clear             # Clear terminal
cluster-info      # Show detailed cluster information  
deps              # Show dependency status
esc               # Switch to cluster selection
```

#### kubectl Commands
All standard kubectl commands work seamlessly:
```bash
get pods                    # List pods
get nodes -o wide          # List nodes with details
apply -f deployment.yaml   # Apply configuration (auto-syncs to Git!)
scale deployment app --replicas=3   # Scale deployment (synced!)
delete pod problematic-pod # Delete resources (synced!)
logs my-pod               # Get pod logs
describe node worker-1    # Describe resources
```

### GitOps Workflow

When ArgoCD is configured for a cluster:

1. **Execute Command**: Run any resource-modifying kubectl command
2. **Auto-Export**: System exports current cluster state to YAML files
3. **Git Commit**: Changes are automatically committed with timestamps
4. **Git Push**: Updates are pushed to the configured repository
5. **ArgoCD Sync**: ArgoCD detects changes and applies them

```bash
[production]$ apply -f new-service.yaml
service/my-service created
✅ Changes synced to Git repository
```

## 🎨 Interface Preview

### Cluster Selection
```
🚀 Kubernetes Orchestrator

Available Clusters:
[1] production-cluster (https://prod.k8s.company.com) 🔍 🚀
[2] staging-cluster (192.168.1.100) 🚀  
[3] development-cluster (dev.k8s.local)
[+] Add New Cluster

↑/↓: navigate • enter: select • q: quit
```

### Terminal Interface
```
🎯 Connected to cluster: production-cluster
Status: 🔍 Prometheus | 🚀 ArgoCD
Terminal Ready - Type 'help' for commands, 'esc' to switch clusters

[production-cluster]$ get pods
NAME                          READY   STATUS    RESTARTS   AGE
nginx-deployment-6b474476c4-8xj2k   1/1     Running   0          2d
nginx-deployment-6b474476c4-h9m3l   1/1     Running   0          2d

[production-cluster]$ apply -f deployment.yaml
deployment.apps/my-app created
✅ Changes synced to Git repository

[production-cluster]$ _
```

## 🔧 Configuration

### Directory Structure
```
~/.kube-orchestrator/
├── configs/                 # Managed kubeconfig files
│   ├── production.yaml
│   ├── staging.yaml
│   └── development.yaml
├── registry.json           # Cluster registry
└── git-repos/              # Cloned Git repositories
    ├── k8s-configs-production/
    └── k8s-configs-staging/
```

### Cluster Registry Format
```json
{
  "clusters": [
    {
      "name": "production-cluster",
      "config_path": "~/.kube-orchestrator/configs/production.yaml",
      "server": "https://prod-k8s.company.com",
      "dns": "prod.k8s.company.com",
      "created_at": "2024-01-15T10:30:45Z",
      "has_prometheus": true,
      "has_argocd": true,
      "git_repo": "https://github.com/company/k8s-configs",
      "git_repo_path": "/tmp/k8s-configs-production"
    }
  ]
}
```

## 🔨 Development

### Building

```bash
# Development build with debug info
make dev

# Production build
make build

# Multi-platform builds
make build-all

# Create release archives
make release
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run security scan
make security

# Lint code
make lint
```

### Code Quality

```bash
# Format code
make fmt

# Run all quality checks
make lint security test
```

## 🐳 Docker Support

```bash
# Build Docker image
make docker-build

# Run in container
docker run -it --rm \
  -v ~/.kube:/root/.kube \
  -v ~/.kube-orchestrator:/root/.kube-orchestrator \
  kube-orchestrator:latest
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`make test lint`)
5. Commit your changes (`git commit -am 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Create a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal UIs
- [kubectl](https://kubernetes.io/docs/reference/kubectl/) - Kubernetes command-line tool
- [ArgoCD](https://argo-cd.readthedocs.io/) - GitOps continuous delivery

## 📞 Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/your-org/kube-orchestrator/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/your-org/kube-orchestrator/discussions)
- 📖 **Documentation**: [Wiki](https://github.com/your-org/kube-orchestrator/wiki)

---

Made with ❤️ for the Kubernetes community