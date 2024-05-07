import FilterListIcon from '@mui/icons-material/FilterList';
import SensorOccupiedIcon from '@mui/icons-material/SensorOccupied';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetConnections } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { IPHistoryTable } from '../component/IPHistoryTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { makeSteamidValidators } from '../util/validator/makeSteamidValidators.ts';

const ipHistorySearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['person_connection_id', 'steam_id', 'created_on', 'server_id']).optional(),
    steam_id: z.string().optional()
});

export const Route = createFileRoute('/_mod/admin/network/iphist')({
    component: AdminNetworkPlayerIPHistory,
    validateSearch: (search) => ipHistorySearchSchema.parse(search)
});

function AdminNetworkPlayerIPHistory() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, rows, sortOrder, sortColumn, steam_id } = Route.useSearch();
    const { data: connections, isLoading } = useQuery({
        queryKey: ['connectionHist', { page, rows, sortOrder, sortColumn, steam_id }],
        queryFn: async () => {
            return await apiGetConnections({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'steam_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: steam_id
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/network/iphist', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: ipHistorySearchSchema
        },
        defaultValues: {
            steam_id: steam_id ?? ''
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/network/iphist',
            search: (prev) => ({ ...prev, source_id: undefined })
        });
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid xs={12}>
                                <Field
                                    name={'steam_id'}
                                    validators={makeSteamidValidators()}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <SteamIDField
                                                state={state}
                                                handleBlur={handleBlur}
                                                handleChange={handleChange}
                                                fullwidth={true}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClear={clear}
                                        />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title="Player IP History" iconLeft={<SensorOccupiedIcon />}>
                    <IPHistoryTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator
                        page={page ?? 0}
                        rows={rows ?? defaultRows}
                        data={connections}
                        path={'/admin/network/iphist'}
                    />
                </ContainerWithHeader>
            </Grid>
            ;
        </Grid>
    );
}
