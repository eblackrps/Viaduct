import { ShieldCheck } from "lucide-react";
import { useState, type FormEvent, type ReactElement } from "react";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import {
	getDashboardAuthPersistence,
	getDashboardAuthSession,
	type DashboardAuthPersistence,
} from "../../runtimeAuth";
import type { AuthBootstrapState } from "./useAuthBootstrap";

interface AuthBootstrapScreenProps {
	auth: AuthBootstrapState;
}

export function AuthBootstrapScreen({ auth }: AuthBootstrapScreenProps) {
	const [mode, setMode] = useState<"service-account" | "tenant">(
		"service-account",
	);
	const [apiKey, setAPIKey] = useState("");
	const [remember, setRemember] = useState(false);
	const [submitting, setSubmitting] = useState(false);
	const [startingLocalSession, setStartingLocalSession] = useState(false);
	const storedSession = getDashboardAuthSession();
	const persistence = getDashboardAuthPersistence();
	const runtimeKeyPresent =
		storedSession.source === "runtime" && storedSession.mode !== "none";
	const errorActions = [
		<button
			key="retry"
			type="button"
			onClick={() => void auth.refresh()}
			className="operator-button-danger"
		>
			Retry validation
		</button>,
		runtimeKeyPresent ? (
			<button
				key="forget"
				type="button"
				onClick={() => auth.signOut()}
				className="operator-button-secondary"
			>
				Forget browser session
			</button>
		) : null,
	].filter((action): action is ReactElement => action !== null);

	async function handleSubmit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setSubmitting(true);
		try {
			await auth.connect(mode, apiKey, remember);
		} finally {
			setSubmitting(false);
		}
	}

	async function handleLocalSession() {
		setStartingLocalSession(true);
		try {
			await auth.connectLocal(remember);
		} finally {
			setStartingLocalSession(false);
		}
	}

	return (
		<main className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
			<div className="mx-auto max-w-[1520px] space-y-6">
				<PageHeader
					eyebrow="Operator Bootstrap"
					title="Connect the Viaduct dashboard"
					description="Start with a scoped service-account session for day-to-day operator work. Use a tenant key only for tenant bootstrap or break-glass administrative recovery."
					badges={[
						{
							label: auth.about
								? `${auth.about.name} ${auth.about.version}`
								: "Build metadata pending",
							tone: "neutral",
						},
						{ label: "Session storage is the default", tone: "info" },
						{
							label: auth.localOperatorAvailable
								? "Local operator bootstrap available"
								: "Runtime authentication",
							tone: "accent",
						},
					]}
				/>

				{auth.status === "checking" ? (
					<LoadingState
						title="Validating dashboard credentials"
						message="The dashboard is confirming the provided service-account or tenant key against the current Viaduct API."
					/>
				) : null}

				{auth.status === "error" && auth.error ? (
					<ErrorState
						title="Authentication failed"
						message={auth.error.message}
						technicalDetails={auth.error.technicalDetails}
						actions={errorActions}
					/>
				) : null}

				<section className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
					<SectionCard
						eyebrow="Authentication"
						title="Runtime credential bootstrap"
						description={
							auth.localOperatorAvailable
								? "This local runtime can issue an operator-scoped session without a pasted key. Service-account and tenant keys remain available for packaged or multi-tenant deployments."
								: "The dashboard accepts runtime credentials so operators can rotate or replace access without rebuilding the frontend. Direct loopback runtime bootstrap is only available on 127.0.0.1, so proxied or remote access should use a service-account or tenant key."
						}
					>
						<form className="space-y-5" onSubmit={handleSubmit}>
							{auth.localOperatorAvailable ? (
								<div className="metric-surface space-y-3">
									<div>
										<p className="operator-kicker">Local lab session</p>
										<p className="mt-2 text-sm leading-6 text-slate-600">
											Use the shipped local runtime to start the workspace-first
											operator flow without pasting a key. The browser stores
											only a non-sensitive session marker while the server keeps
											the effective operator identity.
										</p>
									</div>
									<button
										type="button"
										onClick={() => void handleLocalSession()}
										disabled={startingLocalSession || submitting}
										className="operator-button"
									>
										{startingLocalSession
											? "Starting local session..."
											: "Use local operator session"}
									</button>
								</div>
							) : null}

							<div className="operator-toggle">
								<button
									type="button"
									onClick={() => setMode("service-account")}
									aria-pressed={mode === "service-account"}
									className={`operator-toggle-button ${mode === "service-account" ? "operator-toggle-button-active" : ""}`}
								>
									Service account
								</button>
								<button
									type="button"
									onClick={() => setMode("tenant")}
									aria-pressed={mode === "tenant"}
									className={`operator-toggle-button ${mode === "tenant" ? "operator-toggle-button-active" : ""}`}
								>
									Tenant key
								</button>
							</div>

							<div className="grid gap-4 md:grid-cols-2">
								<div className="md:col-span-2">
									<label className="block">
										<span className="operator-kicker">
											{mode === "service-account"
												? "Service-account key"
												: "Tenant key"}
										</span>
										<input
											type="password"
											value={apiKey}
											onChange={(event) => setAPIKey(event.target.value)}
											className="operator-input mt-2"
											placeholder={
												mode === "service-account"
													? "Paste the operator service-account key"
													: "Paste the tenant bootstrap key"
											}
										/>
									</label>
									<p className="mt-3 text-sm leading-6 text-slate-600">
										{mode === "service-account"
											? "Preferred for the guided workspace flow, background jobs, and exported reports."
											: "Use only when you need tenant bootstrap access or administrative recovery."}
									</p>
								</div>

								<label className="metric-surface flex items-start gap-3 text-sm text-slate-700 md:col-span-2">
									<input
										type="checkbox"
										checked={remember}
										onChange={(event) => setRemember(event.target.checked)}
										className="mt-1 h-4 w-4 rounded border-slate-300"
									/>
									<span>
										<span className="block font-semibold text-ink">
											Remember this browser
										</span>
										<span className="mt-1.5 block leading-6 text-slate-600">
											Persist only a non-sensitive session marker in this
											browser while the server-backed dashboard session stays
											behind an httpOnly cookie. Leave this off for shared
											devices, temporary sessions, or validation labs.
										</span>
									</span>
								</label>
							</div>

							<div className="flex flex-wrap gap-2">
								<button
									type="submit"
									disabled={submitting || apiKey.trim() === ""}
									className={
										auth.localOperatorAvailable
											? "operator-button-secondary"
											: "operator-button"
									}
								>
									{submitting ? "Connecting..." : "Connect dashboard"}
								</button>
								{runtimeKeyPresent ? (
									<button
										type="button"
										onClick={() => auth.signOut()}
										className="operator-button-secondary"
									>
										Forget browser session
									</button>
								) : null}
							</div>
						</form>
					</SectionCard>

					<div className="space-y-6">
						<SectionCard
							eyebrow="Recommended flow"
							title="What operators do next"
							description="The dashboard is optimized for the workspace-first path, not a disconnected collection of demo pages."
						>
							<div className="space-y-3">
								{[
									{
										step: "01",
										title: "Create a workspace",
										body: "Capture the source connection, credential reference, and target assumptions in one persisted record.",
									},
									{
										step: "02",
										title: "Run discovery and inspect",
										body: "Persist snapshots, review workloads, and inspect dependency context before planning.",
									},
									{
										step: "03",
										title: "Simulate, plan, and export",
										body: "Keep readiness results, saved plans, notes, and exported reports attached to the same workspace.",
									},
								].map((item) => (
									<div
										key={item.step}
										className="metric-surface flex items-start gap-3"
									>
										<span className="inline-flex h-11 w-11 items-center justify-center rounded-[18px] bg-ink text-sm font-semibold text-white shadow-[0_10px_20px_rgba(15,23,42,0.18)]">
											{item.step}
										</span>
										<div>
											<p className="text-sm font-semibold text-ink">
												{item.title}
											</p>
											<p className="mt-1 text-sm leading-6 text-slate-600">
												{item.body}
											</p>
										</div>
									</div>
								))}
							</div>
						</SectionCard>

						<SectionCard
							eyebrow="Credential handling"
							title="Storage and recovery"
							description="Keep credential behavior explicit so operators know when a browser session can be trusted and when it should be cleared."
						>
							<div className="grid gap-3">
								<StorageRow
									label="Current source"
									value={describeCurrentSource(
										storedSession.mode,
										storedSession.source,
									)}
								/>
								<StorageRow
									label="Persistence"
									value={describePersistence(persistence)}
								/>
								<StorageRow
									label="Recommended default"
									value="Session-only browser storage"
								/>
							</div>
						</SectionCard>

						<SectionCard
							eyebrow="Current API context"
							title="Operator visibility"
							description="Build metadata and tenant identity appear here as soon as the API responds."
						>
							<div className="flex flex-wrap gap-2">
								{auth.currentTenant ? (
									<>
										<StatusBadge tone="success">
											{auth.currentTenant.name}
										</StatusBadge>
										<StatusBadge tone="info">
											{auth.currentTenant.role}
										</StatusBadge>
										<StatusBadge tone="neutral">
											{auth.currentTenant.auth_method}
										</StatusBadge>
									</>
								) : (
									<StatusBadge tone="neutral">
										Tenant context will appear after validation
									</StatusBadge>
								)}
							</div>
							<div className="mt-5 metric-surface text-sm text-slate-600">
								<div className="flex items-start gap-3">
									<div className="state-icon h-10 w-10 rounded-[16px] bg-white text-slate-600">
										<ShieldCheck className="h-4 w-4" />
									</div>
									<div>
										<p className="font-semibold text-ink">
											Tenant-scoped operator access
										</p>
										<p className="mt-1 leading-6">
											The dashboard uses the same tenant-scoped API and
											persisted backend model as the CLI and packaged operator
											surfaces.
										</p>
									</div>
								</div>
							</div>
						</SectionCard>
					</div>
				</section>
			</div>
		</main>
	);
}

function StorageRow({ label, value }: { label: string; value: string }) {
	return (
		<div className="metric-surface">
			<p className="operator-kicker">{label}</p>
			<p className="mt-2 text-sm font-semibold leading-6 text-ink">{value}</p>
		</div>
	);
}

function describeCurrentSource(
	mode: "local" | "tenant" | "service-account" | "none",
	source: "runtime" | "environment" | "none",
): string {
	if (mode === "none" || source === "none") {
		return "No credential configured";
	}
	if (mode === "local") {
		return "Local operator session issued by the runtime";
	}
	if (mode === "service-account") {
		return source === "runtime"
			? "Service-account key entered at runtime"
			: "Service-account key provided by environment";
	}
	return source === "runtime"
		? "Tenant key entered at runtime"
		: "Tenant key provided by environment";
}

function describePersistence(persistence: DashboardAuthPersistence): string {
	switch (persistence) {
		case "session":
			return "Session marker stored only for the current browser session";
		case "local":
			return "Non-sensitive session marker stored locally until removed";
		case "environment":
			return "Provided by environment configuration";
		default:
			return "Nothing stored yet";
	}
}
