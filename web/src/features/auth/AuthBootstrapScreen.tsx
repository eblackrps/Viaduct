import { ShieldCheck } from "lucide-react";
import { useEffect, useState, type FormEvent, type ReactElement } from "react";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { getDashboardAuthSession } from "../../runtimeAuth";
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
	const [showKeyForm, setShowKeyForm] = useState(false);
	const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

	useEffect(() => {
		if (!auth.localOperatorAvailable) {
			setShowKeyForm(true);
		}
	}, [auth.localOperatorAvailable]);

	const storedSession = getDashboardAuthSession();
	const runtimeKeyPresent =
		storedSession.source === "runtime" && storedSession.mode !== "none";
	const errorActions = [
		<button
			key="retry"
			type="button"
			onClick={() => void auth.refresh()}
			className="operator-button-danger"
		>
			Try again
		</button>,
		runtimeKeyPresent ? (
			<button
				key="forget"
				type="button"
				onClick={() => auth.signOut()}
				className="operator-button-secondary"
			>
				Sign out on this browser
			</button>
		) : null,
	].filter((action): action is ReactElement => action !== null);

	async function handleSubmit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		const submittedKey = apiKey.trim();
		if (submittedKey === "") {
			return;
		}

		setSubmitting(true);
		setAPIKey("");
		try {
			await auth.connect(mode, submittedKey, remember);
		} catch {
			setAPIKey(submittedKey);
		} finally {
			setSubmitting(false);
		}
	}

	async function handleLocalSession() {
		setStartingLocalSession(true);
		try {
			await auth.connectLocal(remember);
		} catch {
			return;
		} finally {
			setStartingLocalSession(false);
		}
	}

	function toggleAdvancedOptions() {
		setShowAdvancedOptions((current) => {
			if (current && mode === "tenant") {
				setMode("service-account");
			}
			return !current;
		});
	}

	const keyModeTitle =
		mode === "service-account"
			? "Service account key"
			: "Tenant key (advanced)";
	const keyModeDescription =
		mode === "service-account"
			? "Best for everyday work in shared or packaged environments."
			: "Use only for tenant setup or emergency access.";
	const showStartOptions = auth.status !== "checking" || auth.about !== null;

	return (
		<main className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
			<div className="mx-auto max-w-5xl space-y-6">
				<PageHeader
					eyebrow="Viaduct dashboard"
					title="Get started"
					description={
						auth.localOperatorAvailable
							? "Viaduct can open a local dashboard session on this machine. No tenant key or service account key is required."
							: "Use a service account key to start your dashboard session. Tenant keys are still available for setup or recovery."
					}
					badges={[
						{
							label: auth.about
								? `${auth.about.name} ${auth.about.version}`
								: "Build metadata pending",
							tone: "neutral",
						},
						{
							label:
								auth.status === "checking" && auth.about === null
									? "Checking sign-in options"
									: auth.localOperatorAvailable
										? "Local session ready"
										: "Key sign-in",
							tone:
								auth.status === "checking" && auth.about === null
									? "neutral"
									: auth.localOperatorAvailable
										? "accent"
										: "info",
						},
					]}
				/>

				{auth.status === "checking" ? (
					<LoadingState
						title="Checking your session"
						message="Viaduct is looking for an existing dashboard session and the best available sign-in option for this runtime."
					/>
				) : null}

				{auth.status === "error" && auth.error ? (
					<ErrorState
						title="We couldn't open the dashboard"
						message={auth.error.message}
						technicalDetails={auth.error.technicalDetails}
						actions={errorActions}
					/>
				) : null}

				{showStartOptions ? (
					<SectionCard
						eyebrow="Start here"
						title={
							auth.localOperatorAvailable
								? "Open the local dashboard"
								: "Start with a key"
						}
						description={
							auth.localOperatorAvailable
								? "The local runtime creates a browser session for this machine so you can go straight to assessments."
								: "Paste a service account key to continue. This keeps sign-in simple for normal work."
						}
					>
						<div className="space-y-5">
							{auth.localOperatorAvailable ? (
								<div className="panel-subtle px-5 py-5">
									<div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
										<div className="max-w-2xl space-y-3">
											<p className="operator-kicker">Recommended</p>
											<div className="space-y-2">
												<p className="text-lg font-semibold text-ink">
													Local session
												</p>
												<p className="text-sm leading-6 text-slate-600">
													Viaduct starts the session for this local runtime.
													There is no key to paste and no tenant setup step for
													the default lab.
												</p>
											</div>
											<ul className="space-y-2 text-sm leading-6 text-slate-600">
												<li>No pasted key required.</li>
												<li>Uses the local runtime you already started.</li>
												<li>
													Works best for the default lab and local workflow.
												</li>
											</ul>
										</div>
										<button
											type="button"
											onClick={() => void handleLocalSession()}
											disabled={startingLocalSession || submitting}
											className="operator-button"
										>
											{startingLocalSession
												? "Starting local session..."
												: "Start local session"}
										</button>
									</div>
								</div>
							) : null}

							{!auth.localOperatorAvailable ? (
								<div className="panel-subtle px-5 py-5">
									<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
										<div className="space-y-2">
											<p className="text-base font-semibold text-ink">
												Use a key
											</p>
											<p className="text-sm leading-6 text-slate-600">
												Service account keys are the normal choice for shared or
												packaged environments.
											</p>
										</div>
									</div>

									{showKeyForm ? (
										<form
											id="auth-key-form"
											className="mt-5 space-y-4 border-t border-slate-200/80 pt-5"
											onSubmit={handleSubmit}
										>
											<div className="space-y-3">
												<div className="flex flex-wrap items-center gap-2">
													<button
														type="button"
														onClick={() => setMode("service-account")}
														aria-pressed={mode === "service-account"}
														className={`operator-toggle-button rounded-full border ${
															mode === "service-account"
																? "border-slate-300 bg-white text-ink shadow-[0_8px_18px_rgba(15,23,42,0.08)]"
																: "border-slate-200 bg-transparent text-slate-600"
														} px-3.5 py-2`}
													>
														Service account key
													</button>
													<button
														type="button"
														onClick={toggleAdvancedOptions}
														aria-expanded={showAdvancedOptions}
														className="operator-button-ghost min-h-0 px-3 py-2"
													>
														{showAdvancedOptions
															? "Hide advanced options"
															: "Show advanced options"}
													</button>
												</div>

												{showAdvancedOptions ? (
													<div className="operator-toggle">
														<button
															type="button"
															onClick={() => setMode("service-account")}
															aria-pressed={mode === "service-account"}
															className={`operator-toggle-button ${mode === "service-account" ? "operator-toggle-button-active" : ""}`}
														>
															Service account key
														</button>
														<button
															type="button"
															onClick={() => setMode("tenant")}
															aria-pressed={mode === "tenant"}
															className={`operator-toggle-button ${mode === "tenant" ? "operator-toggle-button-active" : ""}`}
														>
															Tenant key (advanced)
														</button>
													</div>
												) : null}
											</div>

											<div className="grid gap-4">
												<label className="block">
													<span className="operator-kicker">
														Paste your key
													</span>
													<input
														type="password"
														value={apiKey}
														onChange={(event) => setAPIKey(event.target.value)}
														className="operator-input mt-2"
														placeholder={keyModeTitle}
													/>
												</label>
												<div className="metric-surface">
													<p className="text-sm font-semibold text-ink">
														{keyModeTitle}
													</p>
													<p className="mt-1 text-sm leading-6 text-slate-600">
														{keyModeDescription}
													</p>
												</div>
												<RememberSessionCheckbox
													remember={remember}
													onChange={setRemember}
												/>
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
													{submitting ? "Starting session..." : "Start session"}
												</button>
												{runtimeKeyPresent ? (
													<button
														type="button"
														onClick={() => auth.signOut()}
														className="operator-button-secondary"
													>
														Sign out on this browser
													</button>
												) : null}
											</div>
										</form>
									) : null}
								</div>
							) : null}

							{auth.localOperatorAvailable ? (
								<RememberSessionCheckbox
									remember={remember}
									onChange={setRemember}
									className={showKeyForm ? "hidden" : undefined}
								/>
							) : null}

							<div className="metric-surface flex items-start gap-3">
								<div className="state-icon h-10 w-10 rounded-md bg-white text-slate-600">
									<ShieldCheck className="h-4 w-4" />
								</div>
								<div>
									<p className="text-sm font-semibold text-ink">
										What happens next
									</p>
									<p className="mt-1 text-sm leading-6 text-slate-600">
										After sign-in, you land in the assessment workflow to create
										an assessment, run discovery, inspect workloads, and save a
										plan. This browser stores only a non-sensitive session
										marker.
									</p>
								</div>
							</div>
						</div>
					</SectionCard>
				) : null}
			</div>
		</main>
	);
}

function RememberSessionCheckbox({
	remember,
	onChange,
	className,
}: {
	remember: boolean;
	onChange: (nextValue: boolean) => void;
	className?: string;
}) {
	return (
		<label
			className={[
				"metric-surface flex items-start gap-3 text-sm text-slate-700",
				className,
			]
				.filter(Boolean)
				.join(" ")}
		>
			<input
				type="checkbox"
				checked={remember}
				onChange={(event) => onChange(event.target.checked)}
				className="mt-1 h-4 w-4 rounded border-slate-300"
			/>
			<span>
				<span className="block font-semibold text-ink">
					Keep me signed in on this browser
				</span>
				<span className="mt-1.5 block leading-6 text-slate-600">
					Viaduct keeps only a session marker in this browser. If the API
					restarts, sign in again. Leave this off on shared devices.
				</span>
			</span>
		</label>
	);
}
