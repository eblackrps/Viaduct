import type { DriftReport } from "../types";

interface DriftTimelineProps {
	report: DriftReport | null;
}

export function DriftTimeline({ report }: DriftTimelineProps) {
	return (
		<section className="panel p-5">
			<div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
				<div>
					<p className="font-display text-2xl text-ink">Drift Detection</p>
					<p className="text-sm text-slate-500">
						Track workload changes between the baseline snapshot and the latest
						inventory sample.
					</p>
				</div>
				<div className="flex gap-3 text-sm">
					<span className="rounded-full bg-sky-100 px-3 py-1 font-semibold text-sky-700">
						+{report?.added_vms ?? 0}
					</span>
					<span className="rounded-full bg-rose-100 px-3 py-1 font-semibold text-rose-700">
						-{report?.removed_vms ?? 0}
					</span>
					<span className="rounded-full bg-amber-100 px-3 py-1 font-semibold text-amber-700">
						{report?.modified_vms ?? 0} changed
					</span>
				</div>
			</div>

			<div className="mt-5 space-y-3">
				{(report?.events ?? []).map((event, index) => (
					<article
						key={`${event.vm.id}-${event.type}-${index}`}
						className="rounded-2xl bg-slate-50 px-4 py-4"
					>
						<div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
							<div>
								<p className="font-semibold text-ink">{event.vm.name}</p>
								<p className="text-sm text-slate-500">
									{formatEventType(event.type)}{" "}
									{event.field ? `· ${event.field}` : ""}
								</p>
							</div>
							<span
								className={`rounded-full px-3 py-1 text-xs font-semibold ${severityClass(event.severity)}`}
							>
								{event.severity}
							</span>
						</div>
						{hasEventDelta(event.old_value, event.new_value) && (
							<p className="mt-3 text-sm text-slate-600">
								{formatEventValue(event.old_value)} →{" "}
								{formatEventValue(event.new_value)}
							</p>
						)}
					</article>
				))}

				{(report?.events ?? []).length === 0 && (
					<p className="rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
						No drift events are currently recorded.
					</p>
				)}
			</div>
		</section>
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

function severityClass(severity: DriftReport["events"][number]["severity"]) {
	switch (severity) {
		case "critical":
			return "bg-rose-100 text-rose-700";
		case "warning":
			return "bg-amber-100 text-amber-700";
		default:
			return "bg-sky-100 text-sky-700";
	}
}
