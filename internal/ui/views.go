package ui

import (
	"fmt"
	"strings"
)

// renderClusterSelection renders the cluster selection view
func (a *Application) renderClusterSelection() string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		styles.TitleStyle.Render("🚀 Kubernetes Orchestrator"),
		a.list.View(),
		styles.InfoStyle.Render("↑/↓: navigate • enter: select • q: quit"))
}
// renderAddCluster renders the add cluster form
func (a *Application) renderAddCluster() string {
	var title string
	var instructions string

	switch a.addClusterStep {
	case 0:
		title = "📝 Add New Cluster - Step 1/3"
		instructions = "Enter the cluster name:"
	case 1:
		title = "📝 Add New Cluster - Step 2/3"
		instructions = fmt.Sprintf("Cluster: %s\nEnter public IP or DNS:", a.newCluster.Name)
	case 2:
		title = "📝 Add New Cluster - Step 3/3"
		endpoint := a.newCluster.DNS
		if endpoint == "" {
			endpoint = a.newCluster.PublicIP
		}
		instructions = fmt.Sprintf("Cluster: %s\nEndpoint: %s\nEnter path to kubeconfig file:",
			a.newCluster.Name, endpoint)
	}

	return fmt.Sprintf("\n%s\n\n%s\n\n%s\n\n%s",
		styles.TitleStyle.Render(title),
		instructions,
		a.textInput.View(),
		styles.InfoStyle.Render("enter: next • esc: cancel"))
}

// renderTerminal renders the terminal view
func (a *Application) renderTerminal() string {
	return fmt.Sprintf("%s\n\n%s",
		a.viewport.View(),
		styles.InfoStyle.Render("esc: switch clusters • ctrl+l: clear • ctrl+c: quit"))
}

// renderLoading renders the loading view
func (a *Application) renderLoading() string {
	return fmt.Sprintf("\n%s %s\n\n%s",
		a.spinner.View(),
		styles.LoadingStyle.Render(a.loadingMsg),
		styles.InfoStyle.Render("esc: cancel"))
}

// setupTerminalViewport initializes the terminal viewport
func (a *Application) setupTerminalViewport() {
	a.output = fmt.Sprintf("%s\n%s\n%s\n",
		styles.TitleStyle.Render("🎯 Connected to cluster: "+a.selectedCluster.Name),
		a.getClusterStatusLine(),
		styles.HeaderStyle.Render("Terminal Ready - Type 'help' for commands, 'esc' to switch clusters"))
	a.updateTerminalOutput()
}

// getClusterStatusLine returns the cluster status information
func (a *Application) getClusterStatusLine() string {
	status := []string{}
	if a.selectedCluster.HasPrometheus {
		status = append(status, "🔍 Prometheus")
	}
	if a.selectedCluster.HasArgoCD {
		status = append(status, "🚀 ArgoCD")
	}

	if len(status) > 0 {
		return styles.InfoStyle.Render("Status: " + strings.Join(status, " | "))
	}
	return ""
}

// updateTerminalOutput updates the terminal viewport content
func (a *Application) updateTerminalOutput() {
	content := a.output + "\n" + a.getCurrentPrompt()
	a.viewport.SetContent(content)
	a.viewport.GotoBottom()
}

// updateTerminalPrompt updates just the prompt line
func (a *Application) updateTerminalPrompt() {
	content := a.output + "\n" + a.getCurrentPrompt()
	a.viewport.SetContent(content)
	a.viewport.GotoBottom()
}

// getCurrentPrompt returns the current command prompt
func (a *Application) getCurrentPrompt() string {
	return fmt.Sprintf("%s %s",
		styles.PromptStyle.Render(fmt.Sprintf("[%s]$", a.selectedCluster.Name)),
		a.currentCommand)
}

// getHelpText returns the help text
func (a *Application) getHelpText() string {
	return `🎯 Kubernetes Orchestrator Terminal

Built-in Commands:
  help              - Show this help
  clear             - Clear terminal
  cluster-info      - Show cluster information
  deps              - Show dependency information
  esc               - Switch clusters

Kubectl Commands:
  get pods          - List pods
  get nodes         - List nodes  
  get namespaces    - List namespaces
  describe <resource> <n>  - Describe resource
  logs <pod-name>   - Get pod logs
  apply -f <file>   - Apply resource from file
  create <resource> - Create resource
  delete <resource> <n> - Delete resource

💡 Any kubectl command will work and be executed on the selected cluster.
📤 Resource modifications are automatically synced to Git (if ArgoCD is configured).

Keyboard Shortcuts:
  Ctrl+L  - Clear terminal
  Esc     - Switch clusters
  Ctrl+C  - Quit application

🔧 System Requirements:
  • kubectl - Kubernetes command-line tool
  • git - Version control system

All dependencies are verified on startup.`
}

// getClusterInfo returns detailed cluster information
func (a *Application) getClusterInfo() string {
	info := fmt.Sprintf(`🔍 Cluster Information:
  Name: %s
  Server: %s`, a.selectedCluster.Name, a.selectedCluster.Server)

	if a.selectedCluster.PublicIP != "" {
		info += fmt.Sprintf("\n  Public IP: %s", a.selectedCluster.PublicIP)
	}
	if a.selectedCluster.DNS != "" {
		info += fmt.Sprintf("\n  DNS: %s", a.selectedCluster.DNS)
	}

	info += fmt.Sprintf(`
  Config: %s
  Added: %s
  Prometheus: %v
  ArgoCD: %v`,
		a.selectedCluster.ConfigPath,
		a.selectedCluster.CreatedAt.Format("2006-01-02 15:04:05"),
		a.selectedCluster.HasPrometheus,
		a.selectedCluster.HasArgoCD)

	if a.selectedCluster.GitRepo != "" {
		info += fmt.Sprintf("\n  Git Repo: %s", a.selectedCluster.GitRepo)
	}

	// Add connection status
	if a.kubectlExecutor != nil {
		if err := a.kubectlExecutor.TestConnection(); err != nil {
			info += fmt.Sprintf("\n  %s", styles.ErrorStyle.Render("❌ Connection: Failed"))
		} else {
			info += fmt.Sprintf("\n  %s", styles.SuccessStyle.Render("✅ Connection: Active"))
		}
	}

	return info
}

// getDependencyInfo returns information about system dependencies
func (a *Application) getDependencyInfo() string {
	info := "🔧 System Dependencies:\n\n"

	// Check kubectl
	if kubectlVersion, err := a.dependencyChecker.GetVersion("kubectl"); err != nil {
		info += styles.ErrorStyle.Render("❌ kubectl: Not available\n")
	} else {
		info += styles.SuccessStyle.Render("✅ kubectl: Available\n")
		info += styles.InfoStyle.Render(fmt.Sprintf("    Version: %s\n", strings.Split(kubectlVersion, "\n")[0]))
	}

	// Check git
	if gitVersion, err := a.dependencyChecker.GetVersion("git"); err != nil {
		info += styles.ErrorStyle.Render("❌ git: Not available\n")
	} else {
		info += styles.SuccessStyle.Render("✅ git: Available\n")
		info += styles.InfoStyle.Render(fmt.Sprintf("    Version: %s\n", gitVersion))
	}

	// Git repository status if available
	if a.gitManager != nil {
		info += "\n📁 Git Repository Status:\n"
		if lastCommit, err := a.gitManager.GetLastCommit(); err != nil {
			info += styles.ErrorStyle.Render("❌ Unable to get repository status\n")
		} else {
			info += styles.SuccessStyle.Render("✅ Repository: Connected\n")
			info += styles.InfoStyle.Render(fmt.Sprintf("    Last commit: %s\n", lastCommit))
		}
	}

	return info
}