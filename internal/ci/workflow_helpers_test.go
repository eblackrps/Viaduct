package ci

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflowDefinition struct {
	On   map[string]any              `yaml:"on"`
	Jobs map[string]workflowJobEntry `yaml:"jobs"`
}

type workflowJobEntry struct {
	Uses  string              `yaml:"uses"`
	With  map[string]any      `yaml:"with"`
	Steps []workflowStepEntry `yaml:"steps"`
}

type workflowStepEntry struct {
	Name string         `yaml:"name"`
	Run  string         `yaml:"run"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
}

func loadWorkflow(t *testing.T, fileName string) workflowDefinition {
	t.Helper()

	workflowPath := filepath.Join("..", "..", ".github", "workflows", fileName)
	payload, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", fileName, err)
	}

	var workflow workflowDefinition
	if err := yaml.Unmarshal(payload, &workflow); err != nil {
		t.Fatalf("yaml.Unmarshal(%s) error = %v", fileName, err)
	}
	return workflow
}

func (workflow workflowDefinition) stepNamed(t *testing.T, jobName, stepName string) workflowStepEntry {
	t.Helper()

	job, ok := workflow.Jobs[jobName]
	if !ok {
		t.Fatalf("workflow missing job %q", jobName)
	}
	for _, step := range job.Steps {
		if step.Name == stepName {
			return step
		}
	}
	t.Fatalf("workflow job %q missing step %q", jobName, stepName)
	return workflowStepEntry{}
}
