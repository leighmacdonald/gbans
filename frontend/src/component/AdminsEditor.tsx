import { Stack } from "@mui/system";
import { useQuery } from "@tanstack/react-query";
import { apiGetSMAdmins, apiGetSMGroupImmunities, apiGetSMGroups, apiGetSMOverrides } from "../api";
import { SMAdminsTable } from "./table/SMAdminsTable";
import { SMGroupsTable } from "./table/SMGroupsTable";
import { SMImmunityTable } from "./table/SMImmunityTable";
import { SMOverridesTable } from "./table/SMOverridesTable";

export function AdminsEditor() {
	const { data: groups, isLoading: isLoadingGroups } = useQuery({
		queryKey: ["serverGroups"],
		queryFn: async () => {
			return await apiGetSMGroups();
		},
	});

	const { data: admins, isLoading: isLoadingAdmins } = useQuery({
		queryKey: ["serverAdmins"],
		queryFn: async () => {
			return await apiGetSMAdmins();
		},
	});

	const { data: overrides, isLoading: isLoadingOverrides } = useQuery({
		queryKey: ["serverOverrides"],
		queryFn: async () => {
			return await apiGetSMOverrides();
		},
	});

	const { data: immunities, isLoading: isLoadingImmunities } = useQuery({
		queryKey: ["serverImmunities"],
		queryFn: async () => {
			return await apiGetSMGroupImmunities();
		},
	});

	return (
		<Stack spacing={2}>
			<SMAdminsTable admins={admins ?? []} groups={groups ?? []} isLoading={isLoadingAdmins || isLoadingGroups} />
			<SMGroupsTable groups={groups ?? []} isLoading={isLoadingGroups} />
			<SMOverridesTable overrides={overrides ?? []} isLoading={isLoadingOverrides} />
			<SMImmunityTable
				immunities={immunities ?? []}
				groups={groups ?? []}
				isLoading={isLoadingImmunities || isLoadingGroups}
			/>
		</Stack>
	);
}
