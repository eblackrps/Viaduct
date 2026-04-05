package lifecycle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

// RecommendationType identifies the category of lifecycle guidance.
type RecommendationType string

const (
	// RecommendationTypePlacement suggests moving a workload for better cost or compliance outcomes.
	RecommendationTypePlacement RecommendationType = "placement"
	// RecommendationTypePolicy suggests remediating a policy violation.
	RecommendationTypePolicy RecommendationType = "policy"
	// RecommendationTypeDrift suggests addressing an observed drift event.
	RecommendationTypeDrift RecommendationType = "drift"
)

// RemediationRecommendation describes an actionable lifecycle recommendation.
type RemediationRecommendation struct {
	// VM is the workload the recommendation applies to.
	VM models.VirtualMachine `json:"vm" yaml:"vm"`
	// Type is the recommendation category.
	Type RecommendationType `json:"type" yaml:"type"`
	// Severity is the operator-facing urgency.
	Severity string `json:"severity" yaml:"severity"`
	// Summary is a concise explanation of the recommendation.
	Summary string `json:"summary" yaml:"summary"`
	// Action is the recommended next action.
	Action string `json:"action" yaml:"action"`
	// TargetPlatform is the proposed destination platform when applicable.
	TargetPlatform models.Platform `json:"target_platform,omitempty" yaml:"target_platform,omitempty"`
	// MonthlySavings estimates monthly savings for placement recommendations.
	MonthlySavings float64 `json:"monthly_savings,omitempty" yaml:"monthly_savings,omitempty"`
	// WaiverEligible reports whether an exception could reasonably be used as a temporary mitigation.
	WaiverEligible bool `json:"waiver_eligible,omitempty" yaml:"waiver_eligible,omitempty"`
	// Sources identifies which lifecycle subsystems produced the recommendation.
	Sources []string `json:"sources,omitempty" yaml:"sources,omitempty"`
}

// RecommendationReport contains generated lifecycle guidance for a fleet.
type RecommendationReport struct {
	// Recommendations contains every actionable recommendation.
	Recommendations []RemediationRecommendation `json:"recommendations" yaml:"recommendations"`
	// GeneratedAt is when recommendations were produced.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
}

// SimulationRequest defines a what-if lifecycle movement simulation.
type SimulationRequest struct {
	// TargetPlatform is the destination platform to simulate.
	TargetPlatform models.Platform `json:"target_platform" yaml:"target_platform"`
	// VMIDs scopes the simulation to specific workloads by ID or name.
	VMIDs []string `json:"vm_ids,omitempty" yaml:"vm_ids,omitempty"`
	// IncludeAll simulates moving the entire fleet.
	IncludeAll bool `json:"include_all,omitempty" yaml:"include_all,omitempty"`
	// Waivers applies temporary policy exceptions during simulation.
	Waivers []PolicyWaiver `json:"waivers,omitempty" yaml:"waivers,omitempty"`
}

// SimulationResult summarizes a what-if lifecycle simulation.
type SimulationResult struct {
	// TargetPlatform is the simulated destination platform.
	TargetPlatform models.Platform `json:"target_platform" yaml:"target_platform"`
	// MovedVMs is the number of workloads changed by the simulation.
	MovedVMs int `json:"moved_vms" yaml:"moved_vms"`
	// CurrentMonthlyCost is the current fleet monthly cost estimate.
	CurrentMonthlyCost float64 `json:"current_monthly_cost" yaml:"current_monthly_cost"`
	// SimulatedMonthlyCost is the simulated fleet monthly cost estimate.
	SimulatedMonthlyCost float64 `json:"simulated_monthly_cost" yaml:"simulated_monthly_cost"`
	// MonthlyDelta is the simulated monthly cost change.
	MonthlyDelta float64 `json:"monthly_delta" yaml:"monthly_delta"`
	// PolicyReport contains simulated policy outcomes.
	PolicyReport *PolicyReport `json:"policy_report,omitempty" yaml:"policy_report,omitempty"`
	// RecommendationReport contains generated recommendations for the simulated state.
	RecommendationReport *RecommendationReport `json:"recommendation_report,omitempty" yaml:"recommendation_report,omitempty"`
	// SimulatedInventory is the transformed inventory sample used for the simulation.
	SimulatedInventory *models.DiscoveryResult `json:"simulated_inventory,omitempty" yaml:"simulated_inventory,omitempty"`
}

// RecommendationEngine generates lifecycle recommendations and simulations.
type RecommendationEngine struct {
	costEngine   *CostEngine
	policyEngine *PolicyEngine
}

// NewRecommendationEngine creates a lifecycle recommendation engine.
func NewRecommendationEngine(costEngine *CostEngine, policyEngine *PolicyEngine) *RecommendationEngine {
	return &RecommendationEngine{
		costEngine:   costEngine,
		policyEngine: policyEngine,
	}
}

// Generate produces actionable lifecycle recommendations from inventory, policy, cost, and drift signals.
func (e *RecommendationEngine) Generate(inventory *models.DiscoveryResult, drift *DriftReport, waivers []PolicyWaiver) (*RecommendationReport, error) {
	if e == nil {
		return nil, fmt.Errorf("generate recommendations: engine is nil")
	}
	if inventory == nil {
		return nil, fmt.Errorf("generate recommendations: inventory is nil")
	}

	report := &RecommendationReport{
		Recommendations: make([]RemediationRecommendation, 0),
		GeneratedAt:     time.Now().UTC(),
	}
	seen := make(map[string]struct{})
	addRecommendation := func(recommendation RemediationRecommendation) {
		key := fmt.Sprintf("%s|%s|%s|%s", recommendation.Type, recommendation.VM.ID, recommendation.TargetPlatform, recommendation.Summary)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		report.Recommendations = append(report.Recommendations, recommendation)
	}

	if e.policyEngine != nil {
		policyReport, err := e.policyEngine.EvaluateWithWaivers(inventory, waivers)
		if err != nil {
			return nil, fmt.Errorf("generate recommendations: %w", err)
		}
		for _, violation := range policyReport.Violations {
			addRecommendation(RemediationRecommendation{
				VM:             violation.VM,
				Type:           RecommendationTypePolicy,
				Severity:       string(violation.Severity),
				Summary:        fmt.Sprintf("Policy %s is violated", violation.Policy.Name),
				Action:         violation.Remediation,
				WaiverEligible: violation.Severity != PolicySeverityEnforce,
				Sources:        []string{"policy"},
			})
		}
	}

	if e.costEngine != nil && len(e.costEngine.profiles) > 0 {
		for _, vm := range inventory.VMs {
			comparison, err := e.costEngine.CompareVM(vm)
			if err != nil {
				return nil, fmt.Errorf("generate recommendations: %w", err)
			}
			if comparison.CheapestPlatform == "" || comparison.CheapestPlatform == vm.Platform || comparison.MonthlySavings <= 0 {
				continue
			}
			if !e.placementAllowed(vm, comparison.CheapestPlatform, waivers) {
				continue
			}

			addRecommendation(RemediationRecommendation{
				VM:             vm,
				Type:           RecommendationTypePlacement,
				Severity:       "info",
				Summary:        fmt.Sprintf("Move %s to %s for lower monthly run cost", vm.Name, comparison.CheapestPlatform),
				Action:         fmt.Sprintf("Plan a placement change to %s and validate dependent policy controls.", comparison.CheapestPlatform),
				TargetPlatform: comparison.CheapestPlatform,
				MonthlySavings: comparison.MonthlySavings,
				Sources:        []string{"cost", "policy"},
			})
		}
	}

	if drift != nil {
		for _, event := range drift.Events {
			if event.Severity == DriftSeverityInfo {
				continue
			}
			addRecommendation(RemediationRecommendation{
				VM:             event.VM,
				Type:           RecommendationTypeDrift,
				Severity:       string(event.Severity),
				Summary:        fmt.Sprintf("Investigate %s drift on %s", strings.ReplaceAll(string(event.Type), "_", " "), event.VM.Name),
				Action:         driftAction(event),
				WaiverEligible: event.Type == DriftPolicyViolation,
				Sources:        []string{"drift"},
			})
		}
	}

	sort.Slice(report.Recommendations, func(i, j int) bool {
		if report.Recommendations[i].Severity == report.Recommendations[j].Severity {
			return strings.ToLower(report.Recommendations[i].VM.Name) < strings.ToLower(report.Recommendations[j].VM.Name)
		}
		return recommendationSeverityRank(report.Recommendations[i].Severity) < recommendationSeverityRank(report.Recommendations[j].Severity)
	})

	return report, nil
}

// Simulate runs a what-if fleet movement and returns cost and policy outcomes.
func (e *RecommendationEngine) Simulate(inventory *models.DiscoveryResult, request SimulationRequest) (*SimulationResult, error) {
	if e == nil {
		return nil, fmt.Errorf("simulate lifecycle: engine is nil")
	}
	if inventory == nil {
		return nil, fmt.Errorf("simulate lifecycle: inventory is nil")
	}
	if e.costEngine == nil {
		return nil, fmt.Errorf("simulate lifecycle: cost engine is required")
	}
	if request.TargetPlatform == "" {
		return nil, fmt.Errorf("simulate lifecycle: target platform is required")
	}

	cloned, err := cloneInventoryResult(inventory)
	if err != nil {
		return nil, fmt.Errorf("simulate lifecycle: %w", err)
	}

	selected := selectedVMKeys(request, cloned.VMs)
	if !request.IncludeAll && len(selected) == 0 {
		return nil, fmt.Errorf("simulate lifecycle: no workloads matched simulation scope")
	}

	currentMonthly, err := e.monthlyFleetCost(inventory.VMs)
	if err != nil {
		return nil, fmt.Errorf("simulate lifecycle: %w", err)
	}

	moved := 0
	for index := range cloned.VMs {
		if !request.IncludeAll {
			if _, ok := selected[workloadIDOrName(cloned.VMs[index])]; !ok {
				continue
			}
		}
		cloned.VMs[index].Platform = request.TargetPlatform
		moved++
	}

	simulatedMonthly, err := e.monthlyFleetCost(cloned.VMs)
	if err != nil {
		return nil, fmt.Errorf("simulate lifecycle: %w", err)
	}

	result := &SimulationResult{
		TargetPlatform:       request.TargetPlatform,
		MovedVMs:             moved,
		CurrentMonthlyCost:   currentMonthly,
		SimulatedMonthlyCost: simulatedMonthly,
		MonthlyDelta:         roundCurrency(simulatedMonthly - currentMonthly),
		SimulatedInventory:   cloned,
	}
	if e.policyEngine != nil {
		policyReport, err := e.policyEngine.EvaluateWithWaivers(cloned, request.Waivers)
		if err != nil {
			return nil, fmt.Errorf("simulate lifecycle: %w", err)
		}
		result.PolicyReport = policyReport
	}

	recommendationReport, err := e.Generate(cloned, nil, request.Waivers)
	if err != nil {
		return nil, fmt.Errorf("simulate lifecycle: %w", err)
	}
	result.RecommendationReport = recommendationReport

	return result, nil
}

func (e *RecommendationEngine) placementAllowed(vm models.VirtualMachine, targetPlatform models.Platform, waivers []PolicyWaiver) bool {
	if e.policyEngine == nil {
		return true
	}

	candidate := vm
	candidate.Platform = targetPlatform
	report, err := e.policyEngine.EvaluateWithWaivers(&models.DiscoveryResult{
		Platform:     targetPlatform,
		DiscoveredAt: time.Now().UTC(),
		VMs:          []models.VirtualMachine{candidate},
	}, waivers)
	if err != nil {
		return false
	}
	for _, violation := range report.Violations {
		if violation.Severity == PolicySeverityEnforce {
			return false
		}
	}
	return true
}

func (e *RecommendationEngine) monthlyFleetCost(vms []models.VirtualMachine) (float64, error) {
	total := 0.0
	for _, vm := range vms {
		cost, err := e.costEngine.CalculateVMCost(vm, vm.Platform)
		if err != nil {
			return 0, err
		}
		total += cost.MonthlyTotal
	}
	return roundCurrency(total), nil
}

func cloneInventoryResult(inventory *models.DiscoveryResult) (*models.DiscoveryResult, error) {
	payload, err := json.Marshal(inventory)
	if err != nil {
		return nil, fmt.Errorf("clone inventory result: %w", err)
	}

	var cloned models.DiscoveryResult
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, fmt.Errorf("clone inventory result: %w", err)
	}
	return &cloned, nil
}

func selectedVMKeys(request SimulationRequest, vms []models.VirtualMachine) map[string]struct{} {
	selected := make(map[string]struct{})
	if request.IncludeAll {
		for _, vm := range vms {
			selected[workloadIDOrName(vm)] = struct{}{}
		}
		return selected
	}

	scope := make(map[string]struct{}, len(request.VMIDs))
	for _, item := range request.VMIDs {
		scope[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}

	for _, vm := range vms {
		if _, ok := scope[strings.ToLower(vm.ID)]; ok {
			selected[workloadIDOrName(vm)] = struct{}{}
			continue
		}
		if _, ok := scope[strings.ToLower(vm.Name)]; ok {
			selected[workloadIDOrName(vm)] = struct{}{}
		}
	}
	return selected
}

func workloadIDOrName(vm models.VirtualMachine) string {
	if strings.TrimSpace(vm.ID) != "" {
		return vm.ID
	}
	return vm.Name
}

func driftAction(event DriftEvent) string {
	switch event.Type {
	case DriftPolicyViolation:
		return "Review the policy mismatch, remediate the workload, or apply a temporary waiver with an expiration."
	case DriftRemoved:
		return "Confirm whether the removal was expected and restore the baseline or workload state if needed."
	default:
		return "Investigate the drift cause and reconcile the workload with the approved baseline."
	}
}

func recommendationSeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case string(PolicySeverityEnforce), string(DriftSeverityCritical):
		return 0
	case string(PolicySeverityWarn), string(DriftSeverityWarning):
		return 1
	default:
		return 2
	}
}
