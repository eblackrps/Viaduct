import type { InventoryPlanningDraft } from "../features/inventory/inventoryPlanningDraft";
import { MigrationWorkflow } from "../features/migrations/MigrationWorkflow";

interface MigrationWizardProps {
	planningDraft?: InventoryPlanningDraft | null;
	onPlanningDraftCleared?: () => void;
	onMigrationChange?: () => void;
}

export function MigrationWizard(props: MigrationWizardProps) {
	return <MigrationWorkflow {...props} />;
}
