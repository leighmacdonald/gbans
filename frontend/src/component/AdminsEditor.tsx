import { Stack } from "@mui/system";
import { SMAdminsTable } from "./table/SMAdminsTable";
import { SMGroupsTable } from "./table/SMGroupsTable";
import { SMImmunityTable } from "./table/SMImmunityTable";
import { SMOverridesTable } from "./table/SMOverridesTable";

export function AdminsEditor() {
	return (
		<Stack spacing={2}>
			<SMAdminsTable />
			<SMGroupsTable />
			<SMOverridesTable />
			<SMImmunityTable />
		</Stack>
	);
}
