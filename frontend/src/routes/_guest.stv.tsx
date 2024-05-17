import { useEffect, useState } from 'react';
import { ChevronLeft, CloudDownload } from '@mui/icons-material';
import FilterListIcon from '@mui/icons-material/FilterList';
import FlagIcon from '@mui/icons-material/Flag';
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
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import {
    ColumnFiltersState,
    createColumnHelper,
    getCoreRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    useReactTable
} from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { initColumnFilter, initPagination } from '../types/table.ts';
import { commonTableSearchSchema } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';
import { emptyOrNullString } from '../util/types.ts';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const demosSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'created_on', 'map_name']).optional(),
    map_name: z.string().optional(),
    server_id: z.number().optional(),
    stats: z.string().optional()
});

export const Route = createFileRoute('/_guest/stv')({
    component: STV,
    validateSearch: (search) => demosSchema.parse(search),
    loader: async ({ context }) => {
        const unsorted = await context.queryClient.ensureQueryData({
            queryKey: ['serversSimple'],
            queryFn: apiGetServers
        });
        const demos = await context.queryClient.ensureQueryData({
            queryKey: ['demos'],
            queryFn: async () => {
                return await apiGetDemos();
            }
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
            }),
            demos: demos
        };
    }
});

const columnHelper = createColumnHelper<DemoFile>();

function STV() {
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, stats, map_name, server_id, rows } = Route.useSearch();
    const { servers, demos } = useLoaderData({ from: '/_guest/stv' });
    const { userSteamID, isAuthenticated } = useRouteContext({ from: '/_guest/stv' });

    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(
        initColumnFilter({
            map_name: !emptyOrNullString(map_name) ? map_name : undefined,
            server_id: server_id ?? 0,
            stats: emptyOrNullString(stats) ? stats : undefined
        })
    );
    const [pagination, setPagination] = useState(initPagination(page, rows));

    const { Field, Subscribe, handleSubmit, reset, setFieldValue } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(
                initColumnFilter({
                    map_name: !emptyOrNullString(value.map_name) ? value.map_name : undefined,
                    server_id: value.server_id > 0 ? value.server_id : 0,
                    stats: !emptyOrNullString(stats) ? stats : undefined
                })
            );
            await navigate({ to: '/stv', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: demosSchema
        },
        defaultValues: {
            map_name: map_name ?? '',
            server_id: server_id ?? 0,
            stats: stats ?? ''
        }
    });
    const columns = [
        columnHelper.accessor('server_id', {
            filterFn: (row, _, filterValue) => {
                return filterValue == 0 || row.original.server_id == filterValue;
            },
            enableSorting: true,
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => {
                return (
                    <Button
                        sx={{
                            color: stc(info.row.original.server_name_short)
                        }}
                        onClick={async () => {
                            setFieldValue('server_id', info.getValue() as number);
                            await handleSubmit();
                        }}
                    >
                        {info.row.original.server_name_short}
                    </Button>
                );
            }
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue() as Date)}</Typography>
        }),
        columnHelper.accessor('map_name', {
            filterFn: 'includesString',
            header: () => <TableHeadingCell name={'Map Name'} />,
            cell: (info) => <Typography>{info.getValue() as string}</Typography>
        }),
        columnHelper.accessor('size', {
            header: () => <TableHeadingCell name={'Size'} />,
            cell: (info) => <Typography>{humanFileSize(info.getValue() as number)}</Typography>
        }),
        columnHelper.accessor('stats', {
            enableColumnFilter: true,
            filterFn: (row, _, filterValue) => {
                return Object.keys(row.original.stats).includes(filterValue);
            },
            header: () => <TableHeadingCell name={'Players'} />,
            cell: (info) => <Typography>{Object.keys(Object(info.getValue())).length}</Typography>
        }),

        columnHelper.display({
            id: 'report',
            cell: (info) => (
                <Button
                    disabled={!isAuthenticated()}
                    color={'error'}
                    startIcon={<FlagIcon />}
                    component={RouterLink}
                    to={'/report'}
                    search={{ demo_id: info.row.original.demo_id }}
                >
                    Report
                </Button>
            )
        }),
        columnHelper.display({
            id: 'download',
            cell: (info) => (
                <Button
                    color={'success'}
                    component={Link}
                    href={`/asset/${info.row.original.asset_id}`}
                    startIcon={<CloudDownload />}
                >
                    Download
                </Button>
            )
        })
    ];

    const table = useReactTable({
        data: demos ?? [],
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: getFilteredRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination,
        onColumnFiltersChange: setColumnFilters,
        state: {
            pagination,
            columnFilters
        }
    });

    const clear = async () => {
        setFieldValue('server_id', 0);
        setFieldValue('map_name', '');
        setFieldValue('stats', '');
        await handleSubmit();
        await handleSubmit();
    };

    useEffect(() => {
        // FIXME why does the stats/steam_id field not function correctly?
        // Performs a submit on page load because otherwise the stats filter does not apply.
        handleSubmit();
    }, [handleSubmit]);

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
                                                        {servers.map((s: ServerSimple) => (
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
                                    name={'stats'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3} padding={2}>
                                <Button
                                    fullWidth
                                    disabled={!isAuthenticated()}
                                    startIcon={<ChevronLeft />}
                                    variant={'contained'}
                                    onClick={async () => {
                                        setFieldValue('stats', userSteamID, {
                                            touch: false
                                        });
                                        setColumnFilters((prev) => {
                                            return { ...prev, stats: !emptyOrNullString(stats) ? stats : undefined };
                                        });
                                        await handleSubmit();
                                    }}
                                >
                                    My SteamID
                                </Button>
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
                    <DataTable table={table} isLoading={false} padding={'none'} />
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
                        count={table.getRowCount()}
                        rows={pagination.pageSize}
                        page={pagination.pageIndex}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
