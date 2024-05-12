import { useMemo, useState } from 'react';
import { CloudDownload } from '@mui/icons-material';
import FilterListIcon from '@mui/icons-material/FilterList';
import VideocamIcon from '@mui/icons-material/Videocam';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import {
    ColumnFiltersState,
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, PermissionLevel, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { initColumnFilter, initPagination, initSortOrder, TablePropsAll } from '../types/table.ts';
import { commonTableSearchSchema } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const demosSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'created_on', 'map_name']).optional(),
    map_name: z.string().optional(),
    server_id: z.number().optional(),
    steam_id: z.string().optional(),
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
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, only_mine, sortColumn, steam_id, map_name, server_id, sortOrder, rows } = Route.useSearch();
    const { servers } = useLoaderData({ from: '/_guest/stv' }) as { servers: ServerSimple[] };
    const { hasPermission, userSteamID } = useRouteContext({ from: '/_guest/stv' });
    const [steamIdEnabled, setSteamIdEnabled] = useState(only_mine);
    const [sorting, setSorting] = useState<SortingState>(initSortOrder(sortOrder, sortOrder));
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter());
    const [pagination, setPagination] = useState(initPagination(page, rows));

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
        queryKey: [
            'demos',
            { page, rows, map_name, steam_id, sortOrder, sortColumn, selectedSteamID, server_id, only_mine }
        ],
        queryFn: async () => {
            return await apiGetDemos();
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
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                label={'Steam ID'}
                                                fullwidth={true}
                                                disabled={steamIdEnabled}
                                            />
                                        );
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
                                        <Buttons
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            reset={reset}
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
                <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
                    <STVTable
                        demos={demos ?? []}
                        isLoading={isLoading}
                        pagination={pagination}
                        setPagination={setPagination}
                        sorting={sorting}
                        setSorting={setSorting}
                        columnFilters={columnFilters}
                        setColumnFilters={setColumnFilters}
                    />
                    <PaginatorLocal
                        onRowsChange={(rows) => {
                            setPagination((prev) => {
                                return { ...prev, pageSize: rows };
                            });
                        }}
                        onPageChange={(page) => {
                            setPagination((prev) => {
                                return { ...prev, pageIndex: page };
                            });
                        }}
                        count={demos?.length ?? 0}
                        rows={pagination.pageSize}
                        page={pagination.pageIndex}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<DemoFile>();

export const STVTable = ({
    demos,
    isLoading,
    pagination,
    setPagination,
    setColumnFilters,
    columnFilters,
    sorting,
    setSorting
}: {
    demos: DemoFile[];
    isLoading: boolean;
} & TablePropsAll) => {
    const navigate = useNavigate({ from: Route.fullPath });

    const columns = [
        columnHelper.accessor('server_id', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => {
                return (
                    <Button
                        sx={{
                            color: stc(info.row.original.server_name_short)
                        }}
                        onClick={async () => {
                            await navigate({ search: (prev) => ({ ...prev, server_id: info.getValue() }) });
                        }}
                    >
                        {info.row.original.server_name_short}
                    </Button>
                );
            }
        }),
        columnHelper.accessor('created_on', {
            enableSorting: true,
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('map_name', {
            enableSorting: true,
            header: () => <TableHeadingCell name={'Map Name'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('size', {
            enableSorting: true,
            header: () => <TableHeadingCell name={'Size'} />,
            cell: (info) => <Typography>{humanFileSize(info.getValue())}</Typography>
        }),
        columnHelper.accessor('stats', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Players'} />,
            cell: (info) => <Typography>{Object.keys(info.getValue()).length}</Typography>
        }),
        columnHelper.accessor('asset_id', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Download'} />,
            cell: (info) => (
                <Button
                    component={Link}
                    href={`/asset/${info.getValue()}`}
                    variant={'contained'}
                    startIcon={<CloudDownload />}
                >
                    Download
                </Button>
            )
        })
    ];

    const table = useReactTable({
        data: demos,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        getSortedRowModel: getSortedRowModel(),
        onSortingChange: setSorting,
        onColumnFiltersChange: setColumnFilters,
        state: {
            sorting,
            pagination,
            columnFilters
        }
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
