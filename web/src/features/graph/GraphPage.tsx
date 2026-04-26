import { DependencyGraph } from "../../components/DependencyGraph";
import { PageHeader } from "../../components/primitives/PageHeader";

export function GraphPage() {
	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Analysis"
				title="Dependency graph"
				description="Inspect workload relationships across storage, backup, and network dependencies with search, filtering, and relationship detail."
			/>
			<DependencyGraph />
		</div>
	);
}
