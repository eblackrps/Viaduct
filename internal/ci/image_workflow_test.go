package ci

import (
	"strings"
	"testing"
)

func TestImageWorkflow_UsesDockerMetadataAndBake_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	metaStep := workflow.stepNamed(t, "image", "Extract Docker metadata")
	if metaStep.Uses != "docker/metadata-action@v6.0.0" {
		t.Fatalf("metadata step uses = %q, want docker/metadata-action@v6.0.0", metaStep.Uses)
	}

	buildStep := workflow.stepNamed(t, "image", "Build and push OCI image")
	if buildStep.Uses != "docker/bake-action@v7.1.0" {
		t.Fatalf("build step uses = %q, want docker/bake-action@v7.1.0", buildStep.Uses)
	}
}

func TestImageWorkflow_TagDerivationAndIdentity_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	metaStep := workflow.stepNamed(t, "image", "Extract Docker metadata")
	tags, ok := metaStep.With["tags"].(string)
	if !ok {
		t.Fatalf("metadata tags = %#v, want string", metaStep.With["tags"])
	}
	if !strings.Contains(tags, "type=edge,branch=main") {
		t.Fatalf("metadata tags = %q, want edge tag for main", tags)
	}
	if !strings.Contains(tags, "type=semver,pattern={{version}}") {
		t.Fatalf("metadata tags = %q, want semver version tag", tags)
	}
	if !strings.Contains(tags, "type=semver,pattern={{major}}.{{minor}}") {
		t.Fatalf("metadata tags = %q, want major.minor tag", tags)
	}
	if !strings.Contains(tags, "type=semver,pattern={{major}},enable=${{ startsWith(github.ref, 'refs/tags/v') && !startsWith(github.ref, 'refs/tags/v0.') }}") {
		t.Fatalf("metadata tags = %q, want major tag with major-zero guard", tags)
	}
	if !strings.Contains(tags, "type=raw,value=latest,enable=${{ startsWith(github.ref, 'refs/tags/v') }}") {
		t.Fatalf("metadata tags = %q, want latest tag only for release tags", tags)
	}
	if !strings.Contains(tags, "type=sha,prefix=sha-") {
		t.Fatalf("metadata tags = %q, want sha tag", tags)
	}

	deriveStep := workflow.stepNamed(t, "image", "Derive image metadata")
	if !strings.Contains(deriveStep.Run, `WORKFLOW_REF="${GITHUB_REF}"`) {
		t.Fatalf("derive image metadata run = %q, want workflow ref to follow the actual github ref", deriveStep.Run)
	}
	if strings.Contains(deriveStep.Run, `WORKFLOW_REF="refs/heads/main"`) {
		t.Fatalf("derive image metadata run = %q, should not hardcode main for non-tag runs", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `WORKFLOW_IDENTITY="https://github.com/${GITHUB_REPOSITORY}/.github/workflows/image.yml@${WORKFLOW_REF}"`) {
		t.Fatalf("derive image metadata run = %q, want workflow identity pin", deriveStep.Run)
	}

	verifyStep := workflow.stepNamed(t, "sign", "Verify published signature")
	if !strings.Contains(verifyStep.Run, `--certificate-identity "${WORKFLOW_IDENTITY}"`) {
		t.Fatalf("signature verification run = %q, want certificate identity pin", verifyStep.Run)
	}
}

func TestImageWorkflow_ResolvesDigestFromBakeMetadata_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	resolveStep := workflow.stepNamed(t, "image", "Resolve pushed digest")
	if !strings.Contains(resolveStep.Run, `def find_digest(value):`) {
		t.Fatalf("resolve digest run = %q, want recursive digest lookup helper", resolveStep.Run)
	}
	if !strings.Contains(resolveStep.Run, `descriptor = value.get("containerimage.descriptor")`) {
		t.Fatalf("resolve digest run = %q, want descriptor digest fallback", resolveStep.Run)
	}
	if !strings.Contains(resolveStep.Run, `direct = value.get("containerimage.digest")`) {
		t.Fatalf("resolve digest run = %q, want direct bake metadata digest lookup", resolveStep.Run)
	}
	metadata, ok := resolveStep.Env["BAKE_METADATA"].(string)
	if !ok {
		t.Fatalf("resolve digest env BAKE_METADATA = %#v, want string", resolveStep.Env["BAKE_METADATA"])
	}
	if metadata != "${{ steps.bake.outputs.metadata }}" {
		t.Fatalf("resolve digest env BAKE_METADATA = %q, want ${{ steps.bake.outputs.metadata }}", metadata)
	}
}

func TestImageWorkflow_ScanFailsClosedAndAttestationsExist_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)

	scanStep := workflow.stepNamed(t, "scan", "Run Trivy image scan")
	if !strings.Contains(scanStep.Run, "--exit-code 1") {
		t.Fatalf("scan step run = %q, want --exit-code 1", scanStep.Run)
	}

	if _, ok := workflow.Jobs["sbom"]; !ok {
		t.Fatal("image workflow missing sbom job")
	}
	provenanceJob, ok := workflow.Jobs["provenance"]
	if !ok {
		t.Fatal("image workflow missing provenance job")
	}
	if provenanceJob.Uses != "slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@v2.1.0" {
		t.Fatalf("provenance job uses = %q, want container SLSA generator", provenanceJob.Uses)
	}
}

func loadImageWorkflow(t *testing.T) workflowDefinition {
	t.Helper()
	return loadWorkflow(t, "image.yml")
}
