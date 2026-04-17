package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"gopkg.in/yaml.v3"
)

// PolicyType identifies a lifecycle policy category.
type PolicyType string

const (
	// PlacementPolicy validates platform or placement constraints.
	PlacementPolicy PolicyType = "placement"
	// AffinityPolicy validates grouping or separation rules.
	AffinityPolicy PolicyType = "affinity"
	// CompliancePolicy validates metadata or compliance constraints.
	CompliancePolicy PolicyType = "compliance"
	// CostPolicy validates workload cost thresholds.
	CostPolicy PolicyType = "cost"
)

// PolicySeverity identifies how strongly a policy should be treated.
type PolicySeverity string

const (
	// PolicySeverityEnforce marks a blocking policy.
	PolicySeverityEnforce PolicySeverity = "enforce"
	// PolicySeverityWarn marks a warning policy.
	PolicySeverityWarn PolicySeverity = "warn"
	// PolicySeverityInfo marks an informational policy.
	PolicySeverityInfo PolicySeverity = "info"
)

// Policy contains a named set of lifecycle rules.
type Policy struct {
	// Name is the policy name.
	Name string `json:"name" yaml:"name"`
	// Type is the lifecycle category of the policy.
	Type PolicyType `json:"type" yaml:"type"`
	// Description explains what the policy protects.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Rules are evaluated against each VM in the inventory.
	Rules []PolicyRule `json:"rules" yaml:"rules"`
	// Severity identifies the action level for violations.
	Severity PolicySeverity `json:"severity" yaml:"severity"`
}

// PolicyRule describes a single policy comparison.
type PolicyRule struct {
	// Field is the VM field to evaluate such as platform, cluster, folder, tag:key, or cost.
	Field string `json:"field" yaml:"field"`
	// Operator is the comparison operator.
	Operator string `json:"operator" yaml:"operator"`
	// Value is the value or list of values to compare against.
	Value interface{} `json:"value" yaml:"value"`
	// Message is the human-readable failure explanation.
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// PolicyViolation records a VM and rule that failed evaluation.
type PolicyViolation struct {
	// Policy is the policy that failed.
	Policy Policy `json:"policy" yaml:"policy"`
	// Rule is the specific rule that failed.
	Rule PolicyRule `json:"rule" yaml:"rule"`
	// VM is the workload that violated the rule.
	VM models.VirtualMachine `json:"vm" yaml:"vm"`
	// CurrentValue is the observed value on the VM.
	CurrentValue interface{} `json:"current_value" yaml:"current_value"`
	// Severity is the effective policy severity.
	Severity PolicySeverity `json:"severity" yaml:"severity"`
	// Remediation is a suggested next action.
	Remediation string `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

// PolicyWaiver suppresses a matching policy violation until expiration.
type PolicyWaiver struct {
	// PolicyName identifies the waived policy.
	PolicyName string `json:"policy_name" yaml:"policy_name"`
	// VMID scopes the waiver to a workload identifier when provided.
	VMID string `json:"vm_id,omitempty" yaml:"vm_id,omitempty"`
	// VMName scopes the waiver to a workload name when provided.
	VMName string `json:"vm_name,omitempty" yaml:"vm_name,omitempty"`
	// Reason explains why the waiver was granted.
	Reason string `json:"reason" yaml:"reason"`
	// ExpiresAt is when the waiver stops applying.
	ExpiresAt time.Time `json:"expires_at" yaml:"expires_at"`
}

// PolicyReport summarizes a policy evaluation run.
type PolicyReport struct {
	// Policies are the policies that were evaluated.
	Policies []Policy `json:"policies" yaml:"policies"`
	// Violations are all detected policy failures.
	Violations []PolicyViolation `json:"violations" yaml:"violations"`
	// CompliantVMs is the number of VMs that passed all policies.
	CompliantVMs int `json:"compliant_vms" yaml:"compliant_vms"`
	// NonCompliantVMs is the number of VMs that failed at least one policy.
	NonCompliantVMs int `json:"non_compliant_vms" yaml:"non_compliant_vms"`
	// WaivedViolations is the number of matching violations suppressed by active waivers.
	WaivedViolations int `json:"waived_violations" yaml:"waived_violations"`
	// EvaluatedAt is when the report was generated.
	EvaluatedAt time.Time `json:"evaluated_at" yaml:"evaluated_at"`
}

// PolicyEngine evaluates workload placement, compliance, and cost policies.
type PolicyEngine struct {
	policies   []Policy
	costEngine *CostEngine
}

// NewPolicyEngine creates a policy engine backed by an optional cost engine.
func NewPolicyEngine(costEngine *CostEngine) *PolicyEngine {
	return &PolicyEngine{
		policies:   make([]Policy, 0),
		costEngine: costEngine,
	}
}

// PolicyCount returns the number of loaded lifecycle policies.
func (e *PolicyEngine) PolicyCount() int {
	if e == nil {
		return 0
	}
	return len(e.policies)
}

// LoadPolicies loads policies from a YAML file or every YAML file in a directory.
func (e *PolicyEngine) LoadPolicies(path string) error {
	if e == nil {
		return fmt.Errorf("load policies: engine is nil")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("load policies: stat %s: %w", path, err)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("load policies: read dir %s: %w", path, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
				continue
			}
			if err := e.loadPolicyFile(filepath.Join(path, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	return e.loadPolicyFile(path)
}

// AddPolicy appends a policy to the engine.
func (e *PolicyEngine) AddPolicy(policy Policy) {
	if e == nil {
		return
	}
	if policy.Severity == "" {
		policy.Severity = PolicySeverityWarn
	}
	e.policies = append(e.policies, policy)
}

// Evaluate runs all configured policies against an inventory result.
func (e *PolicyEngine) Evaluate(inventory *models.DiscoveryResult) (*PolicyReport, error) {
	return e.EvaluateWithWaivers(inventory, nil)
}

// EvaluateWithWaivers runs all configured policies while honoring active waivers.
func (e *PolicyEngine) EvaluateWithWaivers(inventory *models.DiscoveryResult, waivers []PolicyWaiver) (*PolicyReport, error) {
	if e == nil {
		return nil, fmt.Errorf("evaluate policies: engine is nil")
	}
	if inventory == nil {
		return nil, fmt.Errorf("evaluate policies: inventory is nil")
	}

	report := &PolicyReport{
		Policies:    append([]Policy(nil), e.policies...),
		Violations:  make([]PolicyViolation, 0),
		EvaluatedAt: time.Now().UTC(),
	}

	nonCompliant := make(map[string]struct{})

	for _, vm := range inventory.VMs {
		for _, policy := range e.policies {
			for _, rule := range policy.Rules {
				passed, currentValue, err := e.evaluateRule(vm, rule)
				if err != nil {
					return nil, err
				}
				if passed {
					continue
				}
				if waiver, waived := matchingWaiver(policy, vm, waivers, report.EvaluatedAt); waived {
					report.WaivedViolations++
					_ = waiver
					continue
				}

				report.Violations = append(report.Violations, PolicyViolation{
					Policy:       policy,
					Rule:         rule,
					VM:           vm,
					CurrentValue: currentValue,
					Severity:     policy.Severity,
					Remediation:  remediationForRule(rule),
				})
				nonCompliant[vmKey(vm)] = struct{}{}
			}
		}
	}

	report.NonCompliantVMs = len(nonCompliant)
	report.CompliantVMs = len(inventory.VMs) - report.NonCompliantVMs
	if report.CompliantVMs < 0 {
		report.CompliantVMs = 0
	}
	return report, nil
}

// Simulate evaluates existing policies plus one candidate policy without mutating the engine.
func (e *PolicyEngine) Simulate(inventory *models.DiscoveryResult, newPolicy Policy) (*PolicyReport, error) {
	if e == nil {
		return nil, fmt.Errorf("simulate policies: engine is nil")
	}

	clone := &PolicyEngine{
		policies:   append([]Policy(nil), e.policies...),
		costEngine: e.costEngine,
	}
	clone.AddPolicy(newPolicy)
	return clone.EvaluateWithWaivers(inventory, nil)
}

func (e *PolicyEngine) loadPolicyFile(path string) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load policies: read %s: %w", path, err)
	}

	var bundle struct {
		Policies []Policy `yaml:"policies"`
	}
	if err := yaml.Unmarshal(payload, &bundle); err == nil && len(bundle.Policies) > 0 {
		for _, policy := range bundle.Policies {
			e.AddPolicy(policy)
		}
		return nil
	}

	var policy Policy
	if err := yaml.Unmarshal(payload, &policy); err != nil {
		return fmt.Errorf("load policies: decode %s: %w", path, err)
	}
	e.AddPolicy(policy)
	return nil
}

func (e *PolicyEngine) evaluateRule(vm models.VirtualMachine, rule PolicyRule) (bool, interface{}, error) {
	currentValue, err := valueForField(vm, rule.Field, e.costEngine)
	if err != nil {
		return false, nil, err
	}

	passed, err := compareValues(currentValue, rule.Operator, rule.Value)
	if err != nil {
		return false, currentValue, err
	}

	return passed, currentValue, nil
}

func valueForField(vm models.VirtualMachine, field string, costEngine *CostEngine) (interface{}, error) {
	switch {
	case field == "platform":
		return string(vm.Platform), nil
	case field == "cluster":
		return vm.Cluster, nil
	case field == "folder":
		return vm.Folder, nil
	case field == "cost":
		if costEngine == nil {
			return nil, fmt.Errorf("value for field cost: cost engine is required")
		}
		cost, err := costEngine.CalculateVMCost(vm, vm.Platform)
		if err != nil {
			return nil, err
		}
		return cost.MonthlyTotal, nil
	case strings.HasPrefix(field, "tag:"):
		key := strings.TrimPrefix(field, "tag:")
		if vm.Tags == nil {
			return "", nil
		}
		return vm.Tags[key], nil
	default:
		return nil, fmt.Errorf("value for field %q: unsupported policy field", field)
	}
}

func compareValues(currentValue interface{}, operator string, expectedValue interface{}) (bool, error) {
	switch operator {
	case "equals":
		return normalizeScalar(currentValue) == normalizeScalar(expectedValue), nil
	case "not-equals":
		return normalizeScalar(currentValue) != normalizeScalar(expectedValue), nil
	case "in":
		values := stringList(expectedValue)
		return slices.Contains(values, normalizeScalar(currentValue)), nil
	case "not-in":
		values := stringList(expectedValue)
		return !slices.Contains(values, normalizeScalar(currentValue)), nil
	case "less-than":
		current, err := numericValue(currentValue)
		if err != nil {
			return false, err
		}
		expected, err := numericValue(expectedValue)
		if err != nil {
			return false, err
		}
		return current < expected, nil
	case "greater-than":
		current, err := numericValue(currentValue)
		if err != nil {
			return false, err
		}
		expected, err := numericValue(expectedValue)
		if err != nil {
			return false, err
		}
		return current > expected, nil
	case "matches":
		pattern, ok := expectedValue.(string)
		if !ok {
			return false, fmt.Errorf("compare matches: expected regex pattern string")
		}
		matched, err := regexp.MatchString(pattern, normalizeScalar(currentValue))
		if err != nil {
			return false, fmt.Errorf("compare matches: %w", err)
		}
		return matched, nil
	default:
		return false, fmt.Errorf("compare values: unsupported operator %q", operator)
	}
}

func normalizeScalar(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case models.Platform:
		return string(typed)
	case PolicySeverity:
		return string(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func numericValue(value interface{}) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("numeric value %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("numeric value %T: unsupported type", value)
	}
}

func stringList(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, normalizeScalar(item))
		}
		return items
	case string:
		parts := strings.Split(typed, ",")
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
		return items
	default:
		return []string{normalizeScalar(value)}
	}
}

func remediationForRule(rule PolicyRule) string {
	if rule.Message != "" {
		return rule.Message
	}
	return fmt.Sprintf("Update %s to satisfy %s %v.", rule.Field, rule.Operator, rule.Value)
}

func vmKey(vm models.VirtualMachine) string {
	if vm.SourceRef != "" {
		return vm.SourceRef
	}
	if vm.ID != "" {
		return vm.ID
	}
	return vm.Name
}

func matchingWaiver(policy Policy, vm models.VirtualMachine, waivers []PolicyWaiver, now time.Time) (PolicyWaiver, bool) {
	for _, waiver := range waivers {
		if !strings.EqualFold(strings.TrimSpace(waiver.PolicyName), strings.TrimSpace(policy.Name)) {
			continue
		}
		if !waiver.ExpiresAt.IsZero() && now.After(waiver.ExpiresAt) {
			continue
		}
		if strings.TrimSpace(waiver.VMID) != "" && strings.TrimSpace(waiver.VMID) != strings.TrimSpace(vm.ID) {
			continue
		}
		if strings.TrimSpace(waiver.VMName) != "" && !strings.EqualFold(strings.TrimSpace(waiver.VMName), strings.TrimSpace(vm.Name)) {
			continue
		}
		return waiver, true
	}
	return PolicyWaiver{}, false
}
