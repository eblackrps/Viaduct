package ci

import (
	"os"
	"path/filepath"
	"regexp"
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

	images, ok := metaStep.With["images"].(string)
	if !ok {
		t.Fatalf("metadata images = %#v, want string", metaStep.With["images"])
	}
	if images != "${{ steps.meta_vars.outputs.registry_images }}" {
		t.Fatalf("metadata images = %q, want registry image list derived from metadata step", images)
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
	if !strings.Contains(deriveStep.Run, `REGISTRY_IMAGES="${IMAGE_NAME}"`) {
		t.Fatalf("derive image metadata run = %q, want GHCR as the canonical registry image", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `REGISTRY_IMAGES="${REGISTRY_IMAGES}"$'\n'"${DOCKERHUB_IMAGE_NAME}"`) {
		t.Fatalf("derive image metadata run = %q, want Docker Hub mirror added to the metadata image list", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `if [[ -n "${DOCKERHUB_USERNAME}" && -n "${DOCKERHUB_TOKEN}" ]]; then`) {
		t.Fatalf("derive image metadata run = %q, want Docker Hub mirroring gated on configured secrets", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `echo "registry_images<<EOF"`) {
		t.Fatalf("derive image metadata run = %q, want registry image list header exported for metadata-action", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `printf '%s\n' "${REGISTRY_IMAGES}"`) {
		t.Fatalf("derive image metadata run = %q, want registry image list payload exported for metadata-action", deriveStep.Run)
	}
	if !strings.Contains(deriveStep.Run, `} >> "${GITHUB_OUTPUT}"`) {
		t.Fatalf("derive image metadata run = %q, want grouped GITHUB_OUTPUT writes for lint-safe output", deriveStep.Run)
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

func TestImageWorkflow_RunsPublishedImageAcceptanceBeforeRelease_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	step := workflow.stepNamed(t, "release-acceptance", "Run published image acceptance smoke")
	if !strings.Contains(step.Run, "go run ./scripts/release_acceptance") {
		t.Fatalf("acceptance run = %q, want release acceptance script", step.Run)
	}
	if !strings.Contains(step.Run, `-image "${IMAGE_NAME}@${DIGEST}"`) {
		t.Fatalf("acceptance run = %q, want published digest image", step.Run)
	}
	if !strings.Contains(step.Run, `-certificate-identity "${WORKFLOW_IDENTITY}"`) {
		t.Fatalf("acceptance run = %q, want cosign identity verification", step.Run)
	}

	releaseJob, ok := workflow.Jobs["release"]
	if !ok {
		t.Fatal("image workflow missing release job")
	}
	if !jobNeeds(releaseJob, "release-acceptance") {
		t.Fatal("release job does not depend on release-acceptance")
	}
}

func TestImageWorkflow_BuildTestCoversSourceReleaseGates_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	goStep := workflow.stepNamed(t, "build-test", "Run Go verification")
	for _, required := range []string{
		"go build ./...",
		"go vet ./...",
		"go test ./... -race -count=1 -timeout 15m",
		"go test -tags soak ./tests/soak/... -count=1",
	} {
		if !strings.Contains(goStep.Run, required) {
			t.Fatalf("go verification run = %q, want %q", goStep.Run, required)
		}
	}

	contractStep := workflow.stepNamed(t, "build-test", "Verify OpenAPI contract")
	for _, required := range []string{
		"go run ./scripts/openapi_generate",
		"git diff --exit-code -- docs/swagger.json",
		"go test ./tests/integration/... -run TestOpenAPISpec_ -count=1",
	} {
		if !strings.Contains(contractStep.Run, required) {
			t.Fatalf("contract run = %q, want %q", contractStep.Run, required)
		}
	}
	if workflow.stepIndex(t, "build-test", "Run Gosec") > workflow.stepIndex(t, "build-test", "Install Web Dependencies") {
		t.Fatal("gosec should run before web dependencies are installed so node_modules is not part of the scan surface")
	}
	if workflow.stepIndex(t, "build-test", "Run Trivy filesystem scan") > workflow.stepIndex(t, "build-test", "Install Web Dependencies") {
		t.Fatal("Trivy filesystem scan should run before web dependencies are installed so node_modules is not part of the scan surface")
	}
}

func TestImageWorkflow_BuildTestCoversRuntimeAndObservabilitySmoke_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	runtimeStep := workflow.stepNamed(t, "build-test", "Run Playwright runtime smoke")
	if !strings.Contains(runtimeStep.Run, "npm run e2e:runtime") {
		t.Fatalf("runtime smoke run = %q, want npm run e2e:runtime", runtimeStep.Run)
	}

	observabilityStep := workflow.stepNamed(t, "build-test", "Run observability smoke")
	for _, required := range []string{
		"make observability-up",
		"./bin/viaduct start --detach --open-browser=false",
		"make observability-validate",
		"make observability-down",
	} {
		if !strings.Contains(observabilityStep.Run, required) {
			t.Fatalf("observability smoke run = %q, want %q", observabilityStep.Run, required)
		}
	}
	if endpoint, _ := observabilityStep.Env["VIADUCT_OTEL_ENDPOINT"].(string); endpoint != "http://127.0.0.1:4318" {
		t.Fatalf("observability endpoint = %q, want local OTLP endpoint", endpoint)
	}
}

func TestImageWorkflow_VerifiesExplicitNativeBundleManifest_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	step := workflow.stepNamed(t, "native-bundles", "Verify expected native bundle artifacts")
	artifactPattern := regexp.MustCompile(`"dist/viaduct_\$\{PACKAGE_VERSION\}_[^"]+\.tar\.gz"`)
	artifacts := artifactPattern.FindAllString(step.Run, -1)
	expectedArtifacts := expectedReleaseArtifacts()
	if len(artifacts) != len(expectedArtifacts) {
		t.Fatalf("artifact manifest = %#v, want %#v", artifacts, expectedArtifacts)
	}
	for index, expected := range expectedArtifacts {
		if artifacts[index] != expected {
			t.Fatalf("artifact manifest[%d] = %q, want %q", index, artifacts[index], expected)
		}
	}
	if !strings.Contains(step.Run, `expected_count=${#expected_artifacts[@]}`) {
		t.Fatal("native bundle verification does not compute expected_count from explicit manifest")
	}
	if !strings.Contains(step.Run, `actual_count=${#actual_artifacts[@]}`) {
		t.Fatal("native bundle verification does not compute actual_count from explicit manifest")
	}
	if !strings.Contains(step.Run, "missing expected release artifact") {
		t.Fatal("native bundle verification does not emit a named missing-artifact error")
	}
	if !strings.Contains(step.Run, "release artifact count") {
		t.Fatal("native bundle verification does not emit a named artifact-count error")
	}
	if !strings.Contains(step.Run, "missing dist/SHA256SUMS") {
		t.Fatal("native bundle verification does not require dist/SHA256SUMS")
	}
}

func TestImageWorkflow_VerifiesNativeBundleChecksumsAgainstPackageMatrix_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	step := workflow.stepNamed(t, "native-bundles", "Verify native bundle checksum manifest")
	if !strings.Contains(step.Run, `sha256sum -c --ignore-missing SHA256SUMS`) {
		t.Fatalf("native bundle checksum run = %q, want SHA256SUMS verification", step.Run)
	}
	for _, binaryPath := range expectedPackageMatrixBinaryPaths(t) {
		if !strings.Contains(step.Run, "sha256sum "+binaryPath) {
			t.Fatalf("native bundle checksum run = %q, want sha256sum for %s", step.Run, binaryPath)
		}
		if !strings.Contains(step.Run, "diff <(sha256sum "+binaryPath+" | cut -d' ' -f1)") {
			t.Fatalf("native bundle checksum run = %q, want checksum diff for %s", step.Run, binaryPath)
		}
	}
	for _, requiredFile := range []string{"release-manifest.json", "dependency-manifest.json", "SHA256SUMS.txt"} {
		if !strings.Contains(step.Run, requiredFile) {
			t.Fatalf("native bundle checksum run = %q, want required bundle file %s", step.Run, requiredFile)
		}
	}
}

func TestImageWorkflow_SignJobAuthenticatesToRegistry_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	authStep := workflow.stepNamed(t, "sign", "Authenticate to GHCR for signing")
	if !strings.Contains(authStep.Run, `docker login ghcr.io`) {
		t.Fatalf("sign auth step run = %q, want docker login to ghcr.io", authStep.Run)
	}
	if !strings.Contains(authStep.Run, `--password-stdin`) {
		t.Fatalf("sign auth step run = %q, want password-stdin login", authStep.Run)
	}
}

func TestImageWorkflow_DockerHubMirrorIsOptional_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadImageWorkflow(t)
	imageJob, ok := workflow.Jobs["image"]
	if !ok {
		t.Fatal("image workflow missing image job")
	}

	foundDockerHubLogin := false
	for _, step := range imageJob.Steps {
		if step.Uses != "docker/login-action@v4.1.0" {
			continue
		}
		registry, _ := step.With["registry"].(string)
		if registry != "docker.io" {
			continue
		}
		foundDockerHubLogin = true
		username, _ := step.With["username"].(string)
		password, _ := step.With["password"].(string)
		if username != "${{ secrets.DOCKERHUB_USERNAME }}" || password != "${{ secrets.DOCKERHUB_TOKEN }}" {
			t.Fatalf("docker hub login credentials = (%q, %q), want Docker Hub secrets", username, password)
		}
	}
	if !foundDockerHubLogin {
		t.Fatal("image workflow missing optional Docker Hub login step")
	}

	workflowPath := filepath.Join("..", "..", ".github", "workflows", "image.yml")
	payload, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(image.yml) error = %v", err)
	}
	if !strings.Contains(string(payload), `if: ${{ steps.meta_vars.outputs.dockerhub_enabled == 'true' }}`) {
		t.Fatal("image workflow does not gate Docker Hub publishing on configured secrets")
	}
}

func TestImageWorkflow_ManualReleaseMirrorBackfill_Expected(t *testing.T) {
	t.Parallel()

	workflowPath := filepath.Join("..", "..", ".github", "workflows", "image.yml")
	payload, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(image.yml) error = %v", err)
	}
	contents := string(payload)

	if !strings.Contains(contents, "mirror_release_tag:") {
		t.Fatal("image workflow missing workflow_dispatch mirror_release_tag input")
	}
	if !strings.Contains(contents, "mirror-existing-release-to-dockerhub:") {
		t.Fatal("image workflow missing manual Docker Hub backfill job")
	}
	if !strings.Contains(contents, `if: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.mirror_release_tag != '' }}`) {
		t.Fatal("image workflow backfill job is not gated on workflow_dispatch mirror input")
	}
	if !strings.Contains(contents, `git fetch --force --tags origin "refs/tags/${RELEASE_TAG}:refs/tags/${RELEASE_TAG}"`) {
		t.Fatal("image workflow backfill job does not fetch the requested release tag")
	}
	if !strings.Contains(contents, `docker buildx imagetools create "${CREATE_ARGS[@]}" "${SOURCE_IMAGE}"`) {
		t.Fatal("image workflow backfill job does not mirror GHCR images to Docker Hub with imagetools create")
	}
	if !strings.Contains(contents, `"${DOCKERHUB_IMAGE_NAME}:sha-${SHORT_SHA}"`) {
		t.Fatal("image workflow backfill job does not mirror the release sha tag")
	}
}

func loadImageWorkflow(t *testing.T) workflowDefinition {
	t.Helper()
	return loadWorkflow(t, "image.yml")
}

func expectedReleaseArtifacts() []string {
	return []string{
		`"dist/viaduct_${PACKAGE_VERSION}_linux_amd64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_linux_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_darwin_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_windows_amd64.tar.gz"`,
	}
}

func expectedPackageMatrixBinaryPaths(t *testing.T) []string {
	t.Helper()

	makefilePath := filepath.Join("..", "..", "Makefile")
	payload, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	content := string(payload)
	return []string{
		extractMakeVariable(t, content, "LINUX_AMD64_BINARY"),
		extractMakeVariable(t, content, "LINUX_ARM64_BINARY"),
	}
}

func extractMakeVariable(t *testing.T, content, name string) string {
	t.Helper()

	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + ` = ([^\r\n]+)$`)
	match := pattern.FindStringSubmatch(content)
	if len(match) != 2 {
		t.Fatalf("Makefile missing variable %s", name)
	}
	return strings.TrimSpace(match[1])
}
