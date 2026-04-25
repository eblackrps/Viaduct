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
	Needs any                 `yaml:"needs"`
	Steps []workflowStepEntry `yaml:"steps"`
}

type workflowStepEntry struct {
	Name string         `yaml:"name"`
	Run  string         `yaml:"run"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
	Env  map[string]any `yaml:"env"`
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

func (workflow workflowDefinition) stepIndex(t *testing.T, jobName, stepName string) int {
	t.Helper()

	job, ok := workflow.Jobs[jobName]
	if !ok {
		t.Fatalf("workflow missing job %q", jobName)
	}
	for index, step := range job.Steps {
		if step.Name == stepName {
			return index
		}
	}
	t.Fatalf("workflow job %q missing step %q", jobName, stepName)
	return -1
}

func jobNeeds(job workflowJobEntry, expected string) bool {
	switch needs := job.Needs.(type) {
	case string:
		return needs == expected
	case []any:
		for _, item := range needs {
			if value, ok := item.(string); ok && value == expected {
				return true
			}
		}
	}
	return false
}
