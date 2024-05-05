import { useMemo, useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import VideocamIcon from '@mui/icons-material/Videocam';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, PermissionLevel, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { makeSteamidValidatorsOptional } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';

const demosSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'created_on', 'map_name']).optional(),
    map_name: z.string().optional(),
    server_id: z.number().optional(),
    steam_id: z.string().optional(),
    orderBy: z.enum(['map_name', 'created_on']).optional(),
    only_mine: z.boolean().optional()
});

export const Route = createFileRoute('/_guest/stv')({
    component: STV,
    validateSearch: (search) => demosSchema.parse(search),
    loader: async ({ context }) => {
        const unsorted = await context.queryClient.ensureQueryData({
            queryKey: ['serversSimple'],
            queryFn: apiGetServers
        });
        return {
            servers: unsorted.sort((a, b) => {
                if (a.server_name > b.server_name) {
                    return 1;
                }
                if (a.server_name < b.server_name) {
                    return -1;
                }
                return 0;
            })
        };
    }
});

function STV() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, only_mine, sortColumn, steam_id, map_name, server_id, sortOrder, rows } = Route.useSearch();
    const { servers } = useLoaderData({ from: '/_guest/stv' }) as { servers: ServerSimple[] };
    const { hasPermission, userSteamID } = useRouteContext({ from: '/_guest/stv' });
    const [steamIdEnabled, setSteamIdEnabled] = useState(only_mine);

    const selectedSteamID = useMemo(() => {
        if (only_mine) {
            return userSteamID;
        } else if (steam_id) {
            return steam_id;
        } else {
            return '';
        }
    }, [only_mine, steam_id, userSteamID]);

    const { data: demos, isLoading } = useQuery({
        queryKey: ['demos', { page, rows, map_name, steam_id, sortOrder, sortColumn, selectedSteamID, server_id, only_mine }],
        queryFn: async () => {
            return await apiGetDemos({
                deleted: false,
                map_name: map_name ?? '',
                server_ids: server_id ? [server_id] : [],
                steam_id: selectedSteamID,
                offset: (page ?? 0) * (rows ?? defaultRows),
                limit: rows ?? defaultRows,
                desc: (sortOrder ?? 'desc') == 'desc',
                order_by: sortColumn ?? 'created_on'
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/stv', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: demosSchema
        },
        defaultValues: {
            map_name: map_name ?? '',
            server_id: server_id ?? 0,
            steam_id: steam_id ?? '',
            only_mine: only_mine != undefined ? only_mine : false
        }
    });

    const clear = async () => {
        await navigate({
            to: '/stv',
            search: (prev) => ({
                ...prev,
                steam_id: undefined,
                page: undefined,
                rows: undefined,
                sortColumn: undefined,
                server_id: undefined,
                map_name: undefined,
                sortOrder: undefined,
                only_mine: undefined
            })
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
                        <Grid container spacing={2} padding={2}>
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'server_id'}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <>
                                                <FormControl fullWidth>
                                                    <InputLabel id="server-select-label">Servers</InputLabel>
                                                    <Select
                                                        fullWidth
                                                        value={state.value}
                                                        label="Servers"
                                                        onChange={(e) => {
                                                            handleChange(Number(e.target.value));
                                                        }}
                                                        onBlur={handleBlur}
                                                    >
                                                        <MenuItem value={0}>All</MenuItem>
                                                        {servers.map((s) => (
                                                            <MenuItem value={s.server_id} key={s.server_id}>
                                                                {s.server_name}
                                                            </MenuItem>
                                                        ))}
                                                    </Select>
                                                </FormControl>
                                            </>
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'map_name'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Map Name'} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'steam_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Steam ID'} fullwidth={true} disabled={steamIdEnabled} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={2}>
                                <Field
                                    name={'only_mine'}
                                    children={(props) => {
                                        return (
                                            <CheckboxSimple
                                                {...props}
                                                disabled={!hasPermission(PermissionLevel.User)}
                                                label={'Only Mine'}
                                                onChange={(v) => setSteamIdEnabled(v)}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons canSubmit={canSubmit} isSubmitting={isSubmitting} reset={reset} onClear={clear} />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
                    <STVTable demos={demos ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} path={'/stv'} data={demos} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<DemoFile>();

export const STVTable = ({ demos, isLoading }: { demos: LazyResult<DemoFile>; isLoading: boolean }) => {
    const navigate = useNavigate({ from: Route.fullPath });

    const columns = [
        columnHelper.accessor('server_id', {
            header: () => <HeadingCell name={'Server'} />,
            cell: (info) => {
                return (
                    <Button
                        sx={{
                            color: stc(demos.data[info.row.index].server_name_short)
                        }}
                        onClick={async () => {
                            await navigate({ search: (prev) => ({ ...prev, server_id: info.getValue() }) });
                        }}
                    >
                        {demos.data[info.row.index].server_name_short}
                    </Button>
                );
            }
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('map_name', {
            header: () => <HeadingCell name={'Map Name'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('size', {
            header: () => <HeadingCell name={'Size'} />,
            cell: (info) => <Typography>{humanFileSize(info.getValue())}</Typography>
        }),
        columnHelper.accessor('downloads', {
            header: () => <HeadingCell name={'DL'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('asset.asset_id', {
            header: () => <HeadingCell name={'Links'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        })
    ];

    const table = useReactTable({
        data: demos.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
