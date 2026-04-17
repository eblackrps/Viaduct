import type { DriftReport } from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatCard } from "./primitives/StatCard";
import { StatusBadge } from "./primitives/StatusBadge";

interface DriftTimelineProps {
	report: DriftReport | null;
}

export function DriftTimeline({ report }: DriftTimelineProps) {
	return (
		<SectionCard
			title="Drift detection"
			description="Track workload changes between the baseline snapshot and the latest inventory sample."
		>
			<div className="grid gap-3 md:grid-cols-3">
				<StatCard label="Added" value={`+${report?.added_vms ?? 0}`} />
				<StatCard label="Removed" value={`-${report?.removed_vms ?? 0}`} />
				<StatCard label="Changed" value={String(report?.modified_vms ?? 0)} />
			</div>

			<div className="mt-5 space-y-3">
				{(report?.events ?? []).map((event, index) => (
					<article
						key={`${event.vm.id}-${event.type}-${index}`}
						className="list-card"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">{event.vm.name}</p>
								<p className="mt-1 text-sm text-slate-500">
									{formatEventType(event.type)}
									{event.field ? ` • ${event.field}` : ""}
								</p>
							</div>
							<StatusBadge tone={severityTone(event.severity)}>
								{event.severity}
							</StatusBadge>
						</div>
						{hasEventDelta(event.old_value, event.new_value) ? (
							<div className="mt-4 grid gap-3 md:grid-cols-2">
								<StatCard
									label="Previous"
									value={formatEventValue(event.old_value)}
								/>
								<StatCard
									label="Current"
									value={formatEventValue(event.new_value)}
								/>
							</div>
						) : null}
					</article>
				))}

				{(report?.events ?? []).length === 0 ? (
					<InlineNotice
						message="No drift events are currently recorded."
						tone="success"
					/>
				) : null}
			</div>
		</SectionCard>
	);
}

function formatEventType(eventType: DriftReport["events"][number]["type"]) {
	return eventType.replace(/_/g, " ");
}

function formatEventValue(value: unknown): string {
	if (value === undefined || value === null || value === "") {
		return "n/a";
	}
	if (
		typeof value === "string" ||
		typeof value === "number" ||
		typeof value === "boolean"
	) {
		return String(value);
	}
	return JSON.stringify(value) ?? "n/a";
}

function hasEventDelta(oldValue: unknown, newValue: unknown): boolean {
	return oldValue !== undefined || newValue !== undefined;
}

function severityTone(
	severity: DriftReport["events"][number]["severity"],
): "info" | "warning" | "danger" {
	switch (severity) {
		case "critical":
			return "danger";
		case "warning":
			return "warning";
		default:
			return "info";
	}
}
