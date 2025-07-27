package ui

import (
	"fmt"

	"github.com/RaymondAkachi/custom-kub-cli/internal/config"
)

// clusterItem represents a cluster in the selection list
type clusterItem struct {
	cluster config.ClusterInfo
}

func (i *clusterItem) FilterValue() string { return i.cluster.Name }

func (i *clusterItem) Title() string {
	status := ""
	if i.cluster.HasPrometheus {
		status += " 🔍"
	}
	if i.cluster.HasArgoCD {
		status += " 🚀"
	}
	return fmt.Sprintf("%s%s", i.cluster.Name, status)
}

func (i *clusterItem) Description() string {
	endpoint := i.cluster.Server
	if i.cluster.DNS != "" {
		endpoint = i.cluster.DNS
	} else if i.cluster.PublicIP != "" {
		endpoint = i.cluster.PublicIP
	}
	return endpoint
}

// addClusterItem represents the "add new cluster" option
type addClusterItem struct{}

func (i *addClusterItem) FilterValue() string { return "add new cluster" }
func (i *addClusterItem) Title() string       { return "➕ Add New Cluster" }
func (i *addClusterItem) Description() string { return "Add a new Kubernetes cluster" }