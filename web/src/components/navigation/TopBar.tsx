import { Clock3, KeyRound, LogOut, RefreshCcw } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../../app/navigation";
import { StatusBadge } from "../primitives/StatusBadge";

interface TopBarProps {
	currentPath: AppRoutePath;
	tenantId?: string;
	lastDiscoveryAt?: string;
	refreshing: boolean;
	onRefresh: () => void | Promise<void>;
	authSummary?: {
		modeLabel: string;
		persistenceLabel: string;
	};
	onSignOut?: (() => void) | undefined;
}

export function TopBar({
	currentPath,
	tenantId,
	lastDiscoveryAt,
	refreshing,
	onRefresh,
	authSummary,
	onSignOut,
}: TopBarProps) {
	const currentItem = navigationGroups
		.flatMap((group) => group.items)
		.find((item) => item.path === currentPath);
	const currentGroup = navigationGroups.find((group) =>
		group.items.some((item) => item.path === currentPath),
	);

	return (
		<div className="panel px-5 py-4 lg:px-6">
			<div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
				<div className="min-w-0 space-y-3">
					<div className="flex flex-wrap items-center gap-2">
						<StatusBadge tone="accent">
							{tenantId ? `Tenant ${tenantId}` : "Default tenant"}
						</StatusBadge>
						{currentGroup ? (
							<StatusBadge tone="neutral">{currentGroup.label}</StatusBadge>
						) : null}
						{currentItem ? (
							<StatusBadge tone="info">{currentItem.label}</StatusBadge>
						) : null}
					</div>

					<div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-slate-600">
						<div>
							<p className="operator-kicker">Current surface</p>
							<p className="mt-1 font-semibold text-ink">
								{currentItem?.title ?? "Operator console"}
							</p>
						</div>
						{lastDiscoveryAt ? (
							<div className="flex items-center gap-2">
								<Clock3 className="h-4 w-4 text-slate-400" />
								<span>
									Last discovery{" "}
									{new Date(lastDiscoveryAt).toLocaleString(undefined, {
										dateStyle: "medium",
										timeStyle: "short",
									})}
								</span>
							</div>
						) : null}
					</div>
				</div>

				<div className="flex flex-wrap items-center gap-2 lg:justify-end">
					{authSummary ? (
						<div className="panel-muted inline-flex items-center gap-2 px-3.5 py-2.5 text-xs font-semibold text-slate-600">
							<KeyRound className="h-3.5 w-3.5 text-slate-400" />
							<div>
								<p className="text-[0.72rem] text-slate-500">
									{authSummary.modeLabel}
								</p>
								<p className="mt-0.5 text-slate-700">
									{authSummary.persistenceLabel}
								</p>
							</div>
						</div>
					) : null}
					{onSignOut ? (
						<button
							type="button"
							onClick={onSignOut}
							className="operator-button-secondary px-3.5 py-2.5 text-xs"
						>
							<LogOut className="h-3.5 w-3.5" />
							Sign out
						</button>
					) : null}
					<button
						type="button"
						onClick={() => void onRefresh()}
						disabled={refreshing}
						className="operator-button-secondary px-3.5 py-2.5 text-xs"
					>
						<RefreshCcw
							className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`}
						/>
						{refreshing ? "Refreshing…" : "Refresh"}
					</button>
				</div>
			</div>
		</div>
	);
}
