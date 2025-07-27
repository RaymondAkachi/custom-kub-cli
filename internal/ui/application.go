package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/RaymondAkachi/custom-kub-cli/internal/config"
	"github.com/RaymondAkachi/custom-kub-cli/internal/git"
	"github.com/RaymondAkachi/custom-kub-cli/internal/kubectl"
	"github.com/RaymondAkachi/custom-kub-cli/internal/system"
)

// sessionState represents the current UI state
type sessionState int

const (
	clusterSelectionView sessionState = iota
	addClusterView
	terminalView
	loadingView
)

// Messages for tea.Cmd communication
type clusterSelectedMsg struct{ cluster *config.ClusterInfo }
type clusterAddedMsg struct{ cluster *config.ClusterInfo }
type commandExecutedMsg struct{ output string }
type errorMsg struct{ err error }
type setupCompleteMsg struct{}

// Application represents the main TUI application
type Application struct {
	state           sessionState
	config          *config.Manager
	dependencyChecker *system.DependencyChecker
	selectedCluster *config.ClusterInfo
	kubectlExecutor *kubectl.Executor
	gitManager      *git.Manager

	// UI components
	list         list.Model
	textInput    textinput.Model
	viewport     viewport.Model
	spinner      spinner.Model

	// Add cluster form
	addClusterStep int
	newCluster     config.ClusterInfo

	// Terminal
	commandHistory []string
	currentCommand string
	output         string
	ready          bool
	width          int
	height         int

	// Loading
	loading     bool
	loadingMsg  string
}

// NewApplication creates a new TUI application
func NewApplication(cfg *config.Manager) (*Application, error) {
	// Create list items from clusters
	var items []list.Item
	for _, cluster := range cfg.GetAllClusters() {
		items = append(items, &clusterItem{cluster: cluster})
	}
	items = append(items, &addClusterItem{})

	l := list.New(items, list.NewDefaultDelegate(), 80, 14)
	l.Title = "🚀 Kubernetes Orchestrator"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.Focus()

	vp := viewport.New(80, 20)

	app := &Application{
		state:             clusterSelectionView,
		config:            cfg,
		dependencyChecker: system.NewDependencyChecker(),
		list:              l,
		textInput:         ti,
		viewport:          vp,
		spinner:           s,
		newCluster:        config.ClusterInfo{CreatedAt: time.Now()},
	}

	return app, nil
}

// Init initializes the application
func (a *Application) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, a.spinner.Tick)
}

// Update handles messages and updates the application state
func (a *Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetWidth(msg.Width - 4)
		a.list.SetHeight(msg.Height - 8)
		a.viewport.Width = msg.Width - 4
		a.viewport.Height = msg.Height - 10
		a.ready = true

	case tea.KeyMsg:
		switch a.state {
		case clusterSelectionView:
			return a.updateClusterSelection(msg)
		case addClusterView:
			return a.updateAddCluster(msg)
		case terminalView:
			return a.updateTerminal(msg)
		case loadingView:
			if msg.String() == "esc" {
				a.state = clusterSelectionView
				a.loading = false
			}
		}

	case clusterSelectedMsg:
		return a.handleClusterSelected(msg.cluster)

	case clusterAddedMsg:
		return a.handleClusterAdded(msg.cluster)

	case commandExecutedMsg:
		a.output = msg.output
		a.loading = false
		a.state = terminalView
		a.updateTerminalOutput()
		return a, nil

	case errorMsg:
		a.output = styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		a.loading = false
		a.state = terminalView
		a.updateTerminalOutput()
		return a, nil

	case spinner.TickMsg:
		if a.loading {
			a.spinner, cmd = a.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update components based on current state
	switch a.state {
	case clusterSelectionView:
		a.list, cmd = a.list.Update(msg)
		cmds = append(cmds, cmd)
	case addClusterView:
		a.textInput, cmd = a.textInput.Update(msg)
		cmds = append(cmds, cmd)
	case terminalView:
		a.viewport, cmd = a.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// handleClusterSelected handles cluster selection
func (a *Application) handleClusterSelected(cluster *config.ClusterInfo) (tea.Model, tea.Cmd) {
	a.selectedCluster = cluster
	a.kubectlExecutor = kubectl.NewExecutor(cluster)

	// Initialize git manager if ArgoCD is configured
	if cluster.HasArgoCD {
		var err error
		a.gitManager, err = git.NewManager(cluster, a.kubectlExecutor)
		if err != nil {
			return a, func() tea.Msg {
				return errorMsg{err: fmt.Errorf("failed to initialize git manager: %v", err)}
			}
		}

		// Initialize git repository
		if err := a.gitManager.Initialize(); err != nil {
			return a, func() tea.Msg {
				return errorMsg{err: fmt.Errorf("failed to initialize git repository: %v", err)}
			}
		}
	}

	a.state = terminalView
	a.setupTerminalViewport()
	return a, nil
}

// handleClusterAdded handles new cluster addition
func (a *Application) handleClusterAdded(cluster *config.ClusterInfo) (tea.Model, tea.Cmd) {
	// Refresh the cluster list
	var items []list.Item
	for _, c := range a.config.GetAllClusters() {
		items = append(items, &clusterItem{cluster: c})
	}
	items = append(items, &addClusterItem{})
	a.list.SetItems(items)

	// Select the new cluster
	return a.handleClusterSelected(cluster)
}

// updateClusterSelection handles cluster selection view updates
func (a *Application) updateClusterSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		selectedItem := a.list.SelectedItem()
		if selectedItem == nil {
			return a, nil
		}

		switch item := selectedItem.(type) {
		case *clusterItem:
			return a, func() tea.Msg {
				return clusterSelectedMsg{cluster: &item.cluster}
			}
		case *addClusterItem:
			a.state = addClusterView
			a.addClusterStep = 0
			a.textInput.SetValue("")
			a.textInput.Placeholder = "Enter cluster name..."
			return a, nil
		}

	case "q", "ctrl+c":
		return a, tea.Quit
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

// updateAddCluster handles add cluster form updates
func (a *Application) updateAddCluster(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return a.handleAddClusterStep()
	case "esc":
		a.state = clusterSelectionView
		a.addClusterStep = 0
		return a, nil
	case "ctrl+c":
		return a, tea.Quit
	}

	var cmd tea.Cmd
	a.textInput, cmd = a.textInput.Update(msg)
	return a, cmd
}

// handleAddClusterStep processes each step of the add cluster form
func (a *Application) handleAddClusterStep() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(a.textInput.Value())
	if value == "" {
		return a, nil
	}

	switch a.addClusterStep {
	case 0: // Cluster name
		a.newCluster.Name = value
		a.textInput.SetValue("")
		a.textInput.Placeholder = "Enter public IP or DNS..."
		a.addClusterStep++

	case 1: // Public IP or DNS
		if strings.Count(value, ".") == 3 && !strings.Contains(value, ":") {
			a.newCluster.PublicIP = value
		} else {
			a.newCluster.DNS = value
		}
		a.textInput.SetValue("")
		a.textInput.Placeholder = "Enter path to kubeconfig file..."
		a.addClusterStep++

	case 2: // Kubeconfig path
		a.loading = true
		a.loadingMsg = "Adding cluster and verifying configuration..."
		a.state = loadingView

		return a, func() tea.Msg {
			if err := a.addCluster(value); err != nil {
				return errorMsg{err: err}
			}
			return clusterAddedMsg{cluster: &a.newCluster}
		}
	}

	return a, nil
}

// addCluster adds a new cluster with full validation
func (a *Application) addCluster(configPath string) error {
	// Validate kubeconfig file exists
	if err := a.config.ValidateClusterConfig(&config.ClusterInfo{
		Name:       a.newCluster.Name,
		ConfigPath: configPath,
	}); err != nil {
		return fmt.Errorf("invalid cluster configuration: %v", err)
	}

	// Copy kubeconfig to managed directory
	destPath, err := a.config.CopyKubeConfig(configPath, a.newCluster.Name)
	if err != nil {
		return fmt.Errorf("failed to copy kubeconfig: %v", err)
	}

	a.newCluster.ConfigPath = destPath

	// Parse kubeconfig to get server information
	kubeConfig, err := a.config.ParseKubeConfig(destPath)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %v", err)
	}

	if len(kubeConfig.Clusters) > 0 {
		a.newCluster.Server = kubeConfig.Clusters[0].Cluster.Server
	}

	// Test connection using dependency checker
	if err := a.dependencyChecker.VerifyKubectlConnection(destPath); err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	// For demo purposes, automatically set up ArgoCD
	// In production, you would prompt the user
	a.newCluster.HasArgoCD = true
	a.newCluster.GitRepo = "https://github.com/example/k8s-configs" // Placeholder
	a.newCluster.GitRepoPath = fmt.Sprintf("/tmp/k8s-configs-%s", a.newCluster.Name)

	// Add cluster to configuration
	if err := a.config.AddCluster(a.newCluster); err != nil {
		return fmt.Errorf("failed to add cluster to configuration: %v", err)
	}

	return nil
}

// updateTerminal handles terminal view updates
func (a *Application) updateTerminal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = clusterSelectionView
		return a, nil
	case "ctrl+c":
		return a, tea.Quit
	case "enter":
		if a.currentCommand != "" {
			return a.executeCommand()
		}
	case "ctrl+l":
		a.output = ""
		a.updateTerminalOutput()
		return a, nil
	default:
		// Handle command input
		switch msg.Type {
		case tea.KeyBackspace:
			if len(a.currentCommand) > 0 {
				a.currentCommand = a.currentCommand[:len(a.currentCommand)-1]
			}
		case tea.KeyRunes:
			a.currentCommand += string(msg.Runes)
		}
		a.updateTerminalPrompt()
	}

	var cmd tea.Cmd
	a.viewport, cmd = a.viewport.Update(msg)
	return a, cmd
}

// executeCommand executes a kubectl command
func (a *Application) executeCommand() (tea.Model, tea.Cmd) {
	if a.currentCommand == "" {
		return a, nil
	}

	command := strings.TrimSpace(a.currentCommand)
	a.commandHistory = append(a.commandHistory, command)

	// Add command to output
	a.output += fmt.Sprintf("%s %s\n",
		styles.PromptStyle.Render(fmt.Sprintf("[%s]$", a.selectedCluster.Name)),
		command)

	a.currentCommand = ""
	a.loading = true
	a.state = loadingView
	a.loadingMsg = "Executing command..."

	return a, func() tea.Msg {
		// Handle built-in commands
		if output := a.handleBuiltinCommand(command); output != "" {
			return commandExecutedMsg{output: output}
		}

		// Execute kubectl command
		output, err := a.kubectlExecutor.ExecuteCommand(command)
		if err != nil {
			return errorMsg{err: err}
		}

		// Check if command modifies resources and sync to git
		if kubectl.IsModifyingCommand(command) && a.gitManager != nil {
			if syncErr := a.gitManager.SyncChanges(""); syncErr != nil {
				output += "\n" + styles.ErrorStyle.Render(fmt.Sprintf("Git sync warning: %v", syncErr))
			} else {
				output += "\n" + styles.SuccessStyle.Render("✅ Changes synced to Git repository")
			}
		}

		return commandExecutedMsg{output: output}
	}
}

// handleBuiltinCommand handles built-in terminal commands
func (a *Application) handleBuiltinCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "help":
		return a.getHelpText()
	case "clear":
		return ""
	case "cluster-info":
		return a.getClusterInfo()
	case "deps":
		return a.getDependencyInfo()
	default:
		return "" // Not a built-in command
	}
}

// View renders the current view
func (a *Application) View() string {
	if !a.ready {
		return "\n  Initializing..."
	}

	switch a.state {
	case clusterSelectionView:
		return a.renderClusterSelection()
	case addClusterView:
		return a.renderAddCluster()
	case terminalView:
		return a.renderTerminal()
	case loadingView:
		return a.renderLoading()
	}

	return ""
}