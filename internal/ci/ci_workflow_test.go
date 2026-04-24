package ci

import (
	"strings"
	"testing"
)

func TestCIWorkflow_WorkflowLintUsesPinnedMakeTarget_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadWorkflow(t, "ci.yml")
	job, ok := workflow.Jobs["workflow-lint"]
	if !ok {
		t.Fatal("CI workflow missing workflow-lint job")
	}

	uses := make([]string, 0, len(job.Steps))
	for _, step := range job.Steps {
		if step.Uses != "" {
			uses = append(uses, step.Uses)
		}
	}
	for _, action := range uses {
		if strings.HasPrefix(action, "rhysd/actionlint@") {
			t.Fatalf("workflow-lint uses %q, want the pinned make workflow-lint target", action)
		}
	}

	step := workflow.stepNamed(t, "workflow-lint", "Run actionlint")
	if strings.TrimSpace(step.Run) != "make workflow-lint" {
		t.Fatalf("workflow-lint run = %q, want make workflow-lint", step.Run)
	}
}
