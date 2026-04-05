package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"gopkg.in/yaml.v3"
)

// FormatTable renders a merged discovery result as a human-readable table.
func FormatTable(result *discovery.MergedResult, verbose bool) string {
	if result == nil || len(result.Sources) == 0 {
		return "No VMs discovered.\n"
	}

	useColor := supportsColor()
	byPlatform := make(map[models.Platform][]models.VirtualMachine)
	for _, source := range result.Sources {
		for _, vm := range source.VMs {
			byPlatform[vm.Platform] = append(byPlatform[vm.Platform], vm)
		}
	}

	platforms := make([]string, 0, len(byPlatform))
	for platform := range byPlatform {
		platforms = append(platforms, string(platform))
	}
	sort.Strings(platforms)

	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)

	for idx, platform := range platforms {
		if idx > 0 {
			fmt.Fprintln(writer)
		}

		fmt.Fprintf(writer, "%s\n", strings.ToUpper(platform))
		if verbose {
			fmt.Fprintln(writer, "NAME\tPLATFORM\tSTATE\tCPU\tMEMORY (MB)\tHOST\tCLUSTER\tGUEST OS\tDISKS\tNICS\tSOURCE REF")
		} else {
			fmt.Fprintln(writer, "NAME\tPLATFORM\tSTATE\tCPU\tMEMORY (MB)\tHOST\tCLUSTER")
		}

		vms := byPlatform[models.Platform(platform)]
		sort.Slice(vms, func(i, j int) bool { return vms[i].Name < vms[j].Name })
		for _, vm := range vms {
			state := string(vm.PowerState)
			if useColor {
				state = colorizePowerState(vm.PowerState)
			}

			if verbose {
				fmt.Fprintf(
					writer,
					"%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%d\t%d\t%s\n",
					vm.Name,
					vm.Platform,
					state,
					vm.CPUCount,
					vm.MemoryMB,
					vm.Host,
					vm.Cluster,
					vm.GuestOS,
					len(vm.Disks),
					len(vm.NICs),
					vm.SourceRef,
				)
				continue
			}

			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%d\t%d\t%s\t%s\n",
				vm.Name,
				vm.Platform,
				state,
				vm.CPUCount,
				vm.MemoryMB,
				vm.Host,
				vm.Cluster,
			)
		}
	}

	_ = writer.Flush()

	memoryGB := float64(result.TotalMemoryMB) / 1024
	fmt.Fprintf(
		&buffer,
		"\nDiscovered %d VMs across %d sources (%.0f vCPU, %.1f GB memory)\n",
		result.TotalVMs,
		len(result.Sources),
		float64(result.TotalCPU),
		memoryGB,
	)

	if len(result.ByPlatform) > 0 {
		platforms = platforms[:0]
		for platform := range result.ByPlatform {
			platforms = append(platforms, string(platform))
		}
		sort.Strings(platforms)
		for _, platform := range platforms {
			summary := result.ByPlatform[models.Platform(platform)]
			fmt.Fprintf(
				&buffer,
				"%s: %d VMs, %d vCPU, %.1f GB memory\n",
				platform,
				summary.VMCount,
				summary.TotalCPU,
				float64(summary.TotalMemoryMB)/1024,
			)
		}
	}

	return buffer.String()
}

// FormatJSON renders a merged discovery result as pretty-printed JSON.
func FormatJSON(result *discovery.MergedResult) (string, error) {
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format JSON: %w", err)
	}

	return string(payload), nil
}

// FormatYAML renders a merged discovery result as YAML.
func FormatYAML(result *discovery.MergedResult) (string, error) {
	payload, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("format YAML: %w", err)
	}

	return string(payload), nil
}

// PrintResult dispatches discovery output to the selected formatter and writes it to stdout.
func PrintResult(format string, result *discovery.MergedResult, verbose bool) error {
	switch format {
	case "table":
		_, err := fmt.Fprint(os.Stdout, FormatTable(result, verbose))
		return err
	case "json":
		payload, err := FormatJSON(result)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, payload)
		return err
	case "yaml":
		payload, err := FormatYAML(result)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, payload)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func supportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}

func colorizePowerState(state models.PowerState) string {
	switch state {
	case models.PowerOn:
		return "\x1b[32mon\x1b[0m"
	case models.PowerOff:
		return "\x1b[31moff\x1b[0m"
	case models.PowerSuspend:
		return "\x1b[33msuspended\x1b[0m"
	default:
		return string(state)
	}
}
