package migrate

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

// MatchWorkloads returns the unique VMs selected by one or more workload selectors.
func MatchWorkloads(vms []models.VirtualMachine, selectors []WorkloadSelector) []models.VirtualMachine {
	matched := make([]models.VirtualMachine, 0)
	seen := make(map[string]struct{})

	for _, vm := range vms {
		if !matchesAnySelector(vm, selectors) {
			continue
		}

		key := vm.ID
		if key == "" {
			key = vm.Name
		}
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		matched = append(matched, vm)
	}

	return matched
}

func matchesAnySelector(vm models.VirtualMachine, selectors []WorkloadSelector) bool {
	for _, selector := range selectors {
		if selectorMatches(vm, selector) {
			return true
		}
	}

	return false
}

func selectorMatches(vm models.VirtualMachine, selector WorkloadSelector) bool {
	match := selector.Match
	if len(match.Exclude) > 0 {
		for _, pattern := range match.Exclude {
			if matchPattern(vm.Name, pattern) {
				return false
			}
		}
	}

	if match.NamePattern != "" && !matchPattern(vm.Name, match.NamePattern) {
		return false
	}
	if !matchTags(vm.Tags, match.Tags) {
		return false
	}
	if match.Folder != "" && !strings.EqualFold(vm.Folder, match.Folder) {
		return false
	}
	if match.PowerState != "" && vm.PowerState != match.PowerState {
		return false
	}

	return true
}

func matchPattern(name, pattern string) bool {
	if strings.HasPrefix(pattern, "regex:") {
		expression, err := regexp.Compile(strings.TrimPrefix(pattern, "regex:"))
		if err != nil {
			return false
		}
		return expression.MatchString(name)
	}

	return matchGlob(name, pattern)
}

func buildWorkloadMigrations(vms []models.VirtualMachine, selectors []WorkloadSelector) []WorkloadMigration {
	planned := make([]WorkloadMigration, 0)
	seen := make(map[string]struct{})

	for _, vm := range vms {
		overrides, ok := mergedOverridesForVM(vm, selectors)
		if !ok {
			continue
		}

		key := vm.ID
		if key == "" {
			key = vm.Name
		}
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		planned = append(planned, WorkloadMigration{
			VM:              vm,
			Phase:           PhasePlan,
			NetworkMappings: copyStringMap(overrides.NetworkMap),
		})
	}

	return planned
}

func mergedOverridesForVM(vm models.VirtualMachine, selectors []WorkloadSelector) (WorkloadOverrides, bool) {
	var (
		overrides WorkloadOverrides
		matched   bool
	)

	for _, selector := range selectors {
		if !selectorMatches(vm, selector) {
			continue
		}

		matched = true
		if overrides.NetworkMap == nil {
			overrides.NetworkMap = make(map[string]string)
		}
		if overrides.StorageMap == nil {
			overrides.StorageMap = make(map[string]string)
		}

		if selector.Overrides.TargetHost != "" {
			overrides.TargetHost = selector.Overrides.TargetHost
		}
		if selector.Overrides.TargetStorage != "" {
			overrides.TargetStorage = selector.Overrides.TargetStorage
		}
		for key, value := range selector.Overrides.NetworkMap {
			overrides.NetworkMap[key] = value
		}
		for key, value := range selector.Overrides.StorageMap {
			overrides.StorageMap[key] = value
		}
		for _, dependency := range selector.Overrides.Dependencies {
			if dependency == "" {
				continue
			}
			if !containsString(overrides.Dependencies, dependency) {
				overrides.Dependencies = append(overrides.Dependencies, dependency)
			}
		}
	}

	return overrides, matched
}

func matchGlob(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

func matchTags(vmTags, selectorTags map[string]string) bool {
	if len(selectorTags) == 0 {
		return true
	}

	if len(vmTags) == 0 {
		return false
	}

	for key, value := range selectorTags {
		if actual, ok := vmTags[key]; !ok || actual != value {
			return false
		}
	}

	return true
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	copied := make(map[string]string, len(input))
	for key, value := range input {
		copied[key] = value
	}

	return copied
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
