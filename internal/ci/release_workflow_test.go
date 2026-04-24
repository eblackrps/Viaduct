package ci

import (
	"strings"
	"testing"
)

func TestReleaseWorkflow_IsGuardOnly_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadReleaseWorkflow(t)
	job, ok := workflow.Jobs["canonical-release-only"]
	if !ok {
		t.Fatal("release workflow missing canonical-release-only guard job")
	}
	if len(job.Steps) != 1 {
		t.Fatalf("release guard job steps = %d, want 1", len(job.Steps))
	}

	step := workflow.stepNamed(t, "canonical-release-only", "Fail With Canonical Workflow Guidance")
	if !strings.Contains(step.Run, ".github/workflows/image.yml") {
		t.Fatalf("guard workflow run = %q, want canonical image workflow guidance", step.Run)
	}
	if !strings.Contains(step.Run, "must not publish images, signatures, SBOMs, provenance, Docker Hub mirrors, or GitHub releases") {
		t.Fatalf("guard workflow run = %q, want duplicate-publishing warning", step.Run)
	}
	if strings.Contains(step.Run, "gh release create") || strings.Contains(step.Run, "docker buildx") || strings.Contains(step.Run, "cosign sign") {
		t.Fatalf("guard workflow run = %q, should not publish release artifacts", step.Run)
	}
}

func TestImageWorkflow_OwnsCanonicalTagRelease_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	releaseJob, ok := workflow.Jobs["release"]
	if !ok {
		t.Fatal("image workflow missing release job")
	}
	if releaseJob.Uses != "" {
		t.Fatalf("release job uses = %q, want inline release job", releaseJob.Uses)
	}

	releaseStep := workflow.stepNamed(t, "release", "Create or update GitHub release")
	if !strings.Contains(releaseStep.Run, `release_notes="docs/releases/${GITHUB_REF_NAME}.md"`) {
		t.Fatalf("release step run = %q, want versioned release notes path", releaseStep.Run)
	}
	if !strings.Contains(releaseStep.Run, `gh release create "${GITHUB_REF_NAME}"`) {
		t.Fatalf("release step run = %q, want GitHub release creation in image workflow", releaseStep.Run)
	}
	if !strings.Contains(releaseStep.Run, `gh release upload "${GITHUB_REF_NAME}" --clobber "${release_assets[@]}"`) {
		t.Fatalf("release step run = %q, want asset upload in image workflow", releaseStep.Run)
	}

	signStep := workflow.stepNamed(t, "native-bundles", "Sign native bundle assets")
	if !strings.Contains(signStep.Run, `--certificate-identity "https://github.com/${GITHUB_REPOSITORY}/.github/workflows/image.yml@${GITHUB_REF}"`) {
		t.Fatalf("native bundle signing run = %q, want image.yml workflow identity", signStep.Run)
	}
}

func loadReleaseWorkflow(t *testing.T) workflowDefinition {
	t.Helper()
	return loadWorkflow(t, "release.yml")
}
