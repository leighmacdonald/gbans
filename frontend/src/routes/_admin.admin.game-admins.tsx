import Stack from '@mui/material/Stack';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { apiGetSMAdmins, apiGetSMGroupImmunities, apiGetSMGroups, apiGetSMOverrides } from '../api';
import { Title } from '../component/Title';
import { SMAdminsTable } from '../component/table/SMAdminsTable.tsx';
import { SMGroupsTable } from '../component/table/SMGroupsTable.tsx';
import { SMImmunityTable } from '../component/table/SMImmunityTable.tsx';
import { SMOverridesTable } from '../component/table/SMOverridesTable.tsx';

export const Route = createFileRoute('/_admin/admin/game-admins')({
    component: AdminsEditor
});

function AdminsEditor() {
    const { data: groups, isLoading: isLoadingGroups } = useQuery({
        queryKey: ['serverGroups'],
        queryFn: async () => {
            return await apiGetSMGroups();
        }
    });

    const { data: admins, isLoading: isLoadingAdmins } = useQuery({
        queryKey: ['serverAdmins'],
        queryFn: async () => {
            return await apiGetSMAdmins();
        }
    });

    const { data: overrides, isLoading: isLoadingOverrides } = useQuery({
        queryKey: ['serverOverrides'],
        queryFn: async () => {
            return await apiGetSMOverrides();
        }
    });

    const { data: immunities, isLoading: isLoadingImmunities } = useQuery({
        queryKey: ['serverImmunities'],
        queryFn: async () => {
            return await apiGetSMGroupImmunities();
        }
    });

    return (
        <>
            <Title>Game Admins</Title>
            <Stack spacing={2}>
                <SMAdminsTable
                    admins={admins ?? []}
                    groups={groups ?? []}
                    isLoading={isLoadingAdmins || isLoadingGroups}
                />
                <SMGroupsTable groups={groups ?? []} isLoading={isLoadingGroups} />
                <SMOverridesTable overrides={overrides ?? []} isLoading={isLoadingOverrides} />
                <SMImmunityTable
                    immunities={immunities ?? []}
                    groups={groups ?? []}
                    isLoading={isLoadingImmunities || isLoadingGroups}
                />
            </Stack>
        </>
    );
}
