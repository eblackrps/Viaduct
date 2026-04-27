import { hasApiKeyConfigured, type ErrorDisplay } from "../../api";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { TenantSummary } from "../../types";
import { useSettingsData } from "./useSettingsData";

interface SettingsPageProps {
	summary: TenantSummary | null;
	authSourceLabel: string;
	authPersistenceLabel: string;
	onForgetRuntimeKey?: (() => void) | undefined;
}

export function SettingsPage({
	summary,
	authSourceLabel,
	authPersistenceLabel,
	onForgetRuntimeKey,
}: SettingsPageProps) {
	const authConfigured = hasApiKeyConfigured();
	const { about, currentTenant, readiness, loading, errors } =
		useSettingsData();
	const supportedPlatforms =
		about?.supported_platforms ?? Object.keys(summary?.platform_counts ?? {});
	const platformCounts = summary?.platform_counts ?? {};
	const openCircuits =
		readiness?.circuit_breakers?.filter(
			(circuit) => circuit.state === "open",
		) ?? [];
	const settingsError = [
		errors.about?.message,
		errors.currentTenant?.message,
		errors.readiness?.message,
	]
		.filter(Boolean)
		.join(" ");
	const settingsErrorDetails = Array.from(
		new Set([
			...(errors.about?.technicalDetails ?? []),
			...(errors.currentTenant?.technicalDetails ?? []),
			...(errors.readiness?.technicalDetails ?? []),
		]),
	);
	const showEmpty = !loading && !about && !currentTenant;

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Settings"
				title="Settings"
				description="Read-only runtime context for the current tenant, sign-in handling, and dashboard settings."
				badges={[
					{
						label: authSourceLabel,
						tone: authConfigured ? "success" : "warning",
					},
					{ label: authPersistenceLabel, tone: "neutral" },
					{
						label: currentTenant?.tenant_id
							? `Tenant ${currentTenant.tenant_id}`
							: summary?.tenant_id
								? `Tenant ${summary.tenant_id}`
								: "No tenant summary",
						tone: "neutral",
					},
				]}
			/>

			{loading && !about && !currentTenant ? (
				<LoadingState
					title="Loading settings"
					message="Retrieving build metadata and tenant context from the Viaduct API."
				/>
			) : null}

			{showEmpty ? (
				settingsError ? (
					<ErrorState
						title="Settings unavailable"
						message={settingsError}
						technicalDetails={settingsErrorDetails}
					/>
				) : (
					<EmptyState
						title="No runtime context available"
						message="The dashboard could not load either build metadata or the current tenant context from the API."
					/>
				)
			) : null}

			<section className="grid gap-5 xl:grid-cols-2">
				<SectionCard
					title="Dashboard connection"
					description="Current dashboard settings for API access."
				>
					{errors.about ? (
						<InlineError error={errors.about} />
					) : (
						<div className="grid gap-3 md:grid-cols-2">
							<StatCard
								label="API base"
								value="/api/v1"
								detail="Uses the Vite proxy in development or the packaged backend in release builds."
							/>
							<StatCard label="Authentication" value={authSourceLabel} />
							<StatCard
								label="Credential persistence"
								value={authPersistenceLabel}
							/>
							<StatCard
								label="Version"
								value={
									about
										? `${about.name} ${about.version} (${about.api_version})`
										: "Unavailable"
								}
							/>
							<StatCard
								label="Build commit"
								value={about?.commit || "Unavailable"}
							/>
							<StatCard
								label="Plugin protocol"
								value={about?.plugin_protocol || "Unavailable"}
							/>
							<StatCard
								label="Store backend"
								value={
									about
										? `${about.store_backend}${about.persistent_store ? " (persistent)" : " (ephemeral)"}`
										: "Unavailable"
								}
							/>
						</div>
					)}
				</SectionCard>

				<SectionCard
					title="Tenant context"
					description="Current summary-level state visible to the dashboard shell."
				>
					{errors.currentTenant ? (
						<InlineError error={errors.currentTenant} />
					) : (
						<div className="grid gap-3 md:grid-cols-2">
							<StatCard
								label="Tenant name"
								value={
									currentTenant?.name ?? summary?.tenant_id ?? "Unavailable"
								}
							/>
							<StatCard
								label="Role"
								value={currentTenant?.role ?? "Unavailable"}
							/>
							<StatCard
								label="Auth method"
								value={currentTenant?.auth_method ?? "Unavailable"}
							/>
							<StatCard
								label="Service account"
								value={
									currentTenant?.service_account_name ?? "Tenant credential"
								}
							/>
							<StatCard
								label="Service accounts"
								value={String(currentTenant?.service_account_count ?? 0)}
							/>
							<StatCard
								label="Workloads"
								value={String(summary?.workload_count ?? 0)}
							/>
							<StatCard
								label="Snapshots"
								value={String(summary?.snapshot_count ?? 0)}
							/>
							<StatCard
								label="Active migrations"
								value={String(summary?.active_migrations ?? 0)}
							/>
							<StatCard
								label="Pending approvals"
								value={String(summary?.pending_approvals ?? 0)}
							/>
						</div>
					)}
				</SectionCard>

				<SectionCard
					title="Sign-in handling"
					description="Session-scoped browser storage is the default path. Remembered sessions still depend on the running API."
				>
					<div className="grid gap-3 md:grid-cols-2">
						<StatCard label="Credential source" value={authSourceLabel} />
						<StatCard label="Persistence" value={authPersistenceLabel} />
						<StatCard
							label="Recommended default"
							value="Session-scoped browser storage"
						/>
						<StatCard
							label="Shared-workstation guidance"
							value="Forget remembered keys after the session ends"
						/>
						<StatCard
							label="API restart behavior"
							value="Sign in again after restart"
						/>
					</div>
					{onForgetRuntimeKey ? (
						<div className="mt-4 flex flex-wrap gap-2">
							<button
								type="button"
								onClick={onForgetRuntimeKey}
								className="operator-button-danger"
							>
								Forget browser key
							</button>
						</div>
					) : (
						<div className="mt-4">
							<InlineNotice
								message="No browser-managed runtime key is active right now. If the dashboard is using an environment-provided key, rotate it through the deployment environment instead."
								tone="neutral"
							/>
						</div>
					)}
				</SectionCard>

				<SectionCard
					title="Permissions and quotas"
					description="Effective permissions and tenant quotas."
					className="xl:col-span-2"
				>
					{currentTenant ? (
						<>
							<div className="flex flex-wrap gap-2">
								{currentTenant.permissions.map((permission) => (
									<StatusBadge key={permission} tone="info">
										{permission}
									</StatusBadge>
								))}
								{currentTenant.permissions.length === 0 ? (
									<StatusBadge tone="neutral">
										No permissions reported
									</StatusBadge>
								) : null}
							</div>
							<div className="mt-4 grid gap-3 md:grid-cols-3">
								<StatCard
									label="Requests/min"
									value={String(currentTenant.quotas?.requests_per_minute ?? 0)}
								/>
								<StatCard
									label="Max snapshots"
									value={String(currentTenant.quotas?.max_snapshots ?? 0)}
								/>
								<StatCard
									label="Max migrations"
									value={String(currentTenant.quotas?.max_migrations ?? 0)}
								/>
							</div>
						</>
					) : (
						<InlineNotice
							message="Tenant permissions and quotas appear here when the current tenant endpoint is available."
							tone="neutral"
						/>
					)}
				</SectionCard>

				<SectionCard
					title="Observed platform coverage"
					description="Supported platforms from the build and workload counts from the current tenant summary."
					className="xl:col-span-2"
				>
					<div className="flex flex-wrap gap-2">
						{supportedPlatforms.map((platform) => (
							<StatusBadge
								key={platform}
								tone={(platformCounts[platform] ?? 0) > 0 ? "info" : "neutral"}
							>
								{platform}: {platformCounts[platform] ?? 0}
							</StatusBadge>
						))}
						{supportedPlatforms.length === 0 ? (
							<StatusBadge tone="neutral">No platforms reported</StatusBadge>
						) : null}
					</div>
					<div className="mt-4">
						<InlineNotice
							message="Build metadata is sourced from `/api/v1/about`, while tenant role, auth method, and effective permissions come from `/api/v1/tenants/current`."
							tone="info"
						/>
					</div>
				</SectionCard>

				<SectionCard
					title="Connector checks"
					description="Credential refs are checked when discovery runs. Open connector circuits appear here."
					className="xl:col-span-2"
				>
					{errors.readiness ? (
						<InlineError error={errors.readiness} />
					) : (
						<div className="space-y-4">
							<div className="grid gap-3 md:grid-cols-3">
								<StatCard
									label="Readiness"
									value={readiness?.status ?? "Unavailable"}
								/>
								<StatCard
									label="Open circuits"
									value={String(openCircuits.length)}
								/>
								<StatCard
									label="Policies"
									value={
										readiness?.policies_loaded === true
											? "Loaded"
											: readiness
												? "Needs review"
												: "Unavailable"
									}
								/>
							</div>
							<div className="flex flex-wrap gap-2">
								{supportedPlatforms.map((platform) => {
									const circuit = readiness?.circuit_breakers?.find(
										(item) => item.platform === platform,
									);
									return (
										<StatusBadge
											key={platform}
											tone={circuit?.state === "open" ? "danger" : "neutral"}
										>
											{platform}: {circuit?.state ?? "ready"}
										</StatusBadge>
									);
								})}
								{supportedPlatforms.length === 0 ? (
									<StatusBadge tone="neutral">
										No platforms reported
									</StatusBadge>
								) : null}
							</div>
							{openCircuits.length > 0 ? (
								<div className="space-y-2">
									{openCircuits.map((circuit) => (
										<InlineNotice
											key={circuit.endpoint}
											tone="warning"
											title={`${circuit.platform} connector unavailable`}
											message={`${circuit.address || circuit.endpoint} will retry after ${circuit.retry_after_seconds ?? 0}s.`}
										/>
									))}
								</div>
							) : (
								<InlineNotice
									message="No open connector circuits are reported right now."
									tone="success"
								/>
							)}
						</div>
					)}
				</SectionCard>
			</section>
		</div>
	);
}

function InlineError({ error }: { error: ErrorDisplay }) {
	return (
		<InlineNotice
			tone="danger"
			title={error.message}
			message={
				error.technicalDetails.length > 0 ? (
					<div className="space-y-1 text-xs">
						{error.technicalDetails.map((detail, index) => (
							<p key={`${detail}-${index}`}>{detail}</p>
						))}
					</div>
				) : (
					""
				)
			}
		/>
	);
}
