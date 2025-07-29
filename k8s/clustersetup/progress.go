// Package k8shard provides automation for setting up Kubernetes clusters.
// progress.go implements the ProgressReporter interface for progress updates.
package clustersetup

import "fmt"

// NewProgressReporter creates a new progress reporter
func NewProgressReporter() ProgressReporter {
	return &consoleProgressReporter{}
}

// consoleProgressReporter is a console-based progress reporter
type consoleProgressReporter struct {
	total   int
	current int
}

func (p *consoleProgressReporter) Start(total int, description string) {
	p.total = total
	p.current = 0
	fmt.Printf("Starting: %s (0/%d)\n", description, total)
}

func (p *consoleProgressReporter) Update(current int, status string) {
	p.current = current
	fmt.Printf("Progress: %s (%d/%d)\n", status, current, p.total)
}

func (p *consoleProgressReporter) Finish(success bool, message string) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	fmt.Printf("%s: %s (%d/%d)\n", status, message, p.total, p.total)
}

func (p *consoleProgressReporter) ReportProgress(step, totalSteps int, phase string) {
	fmt.Printf("Step %d/%d: %s\n", step, totalSteps, phase)
}