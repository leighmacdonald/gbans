import { useState } from 'react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import FilterListIcon from '@mui/icons-material/FilterList';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import Grid from '@mui/material/Grid';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetAnticheatLogs, apiGetServers, Detection, Detections, ServerSimple, StacEntry } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { stringToColour } from '../util/colours.ts';
import { initPagination, initSortOrder, makeCommonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

const schema = z.object({
    ...makeCommonTableSearchSchema([
        'anticheat_id',
        'name',
        'personaname',
        'summary',
        'detection',
        'steam_id',
        'created_on',
        'server_name'
    ]),
    name: z.string().optional(),
    summary: z.string().optional(),
    server_id: z.number().optional(),
    detection: z.string().optional(),
    steam_id: z.string().optional(),
    personaname: z.string().optional()
});

export const Route = createFileRoute('/_mod/admin/anticheat')({
    component: AdminAnticheat,
    validateSearch: (search) => schema.parse(search),
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

const columnHelper = createColumnHelper<StacEntry>();

function AdminAnticheat() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const { servers } = useLoaderData({ from: '/_mod/admin/anticheat' }) as { servers: ServerSimple[] };
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>(
        initSortOrder(search.sortColumn, search.sortOrder, { id: 'created_on', desc: true })
    );
    //const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));
    const [columnVisibility] = useState({
        server_id: true,
        name: true,
        personaname: false,
        steam_id: false
    });

    const { data: logs, isLoading } = useQuery({
        queryKey: ['anticheat', search],
        queryFn: async () => {
            try {
                return await apiGetAnticheatLogs({
                    server_id: search.server_id ?? 0,
                    name: search.name ?? '',
                    summary: search.summary ?? '',
                    steam_id: search.steam_id ?? '',
                    detection: (search.detection ?? '') as Detection,
                    limit: search.pageSize ?? defaultRows,
                    offset: (search.pageIndex ?? 0) * (search.pageSize ?? defaultRows),
                    order_by: 'created_on',
                    desc: (search.sortOrder ?? 'desc') == 'desc'
                });
            } catch {
                return [];
            }
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            //setColumnFilters(initColumnFilter(value));
            await navigate({ to: '/admin/anticheat', search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            name: search.name ?? '',
            summary: search.summary ?? '',
            detection: search.detection ?? '',
            steam_id: search.steam_id ?? '',
            server_id: search.server_id ?? 0
        }
    });

    const clear = async () => {
        //setColumnFilters([]);
        form.reset();
        await navigate({
            to: '/admin/anticheat',
            search: (prev) => ({ ...prev })
        });
    };

    const theme = useTheme();
    const columns = [
        columnHelper.accessor('anticheat_id', {
            header: 'ID',
            size: 50,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('server_id', {
            header: 'Server',
            size: 100,
            cell: (info) => {
                return (
                    <Button
                        sx={{
                            color: stringToColour(info.row.original.server_name, theme.palette.mode)
                        }}
                        onClick={async () => {
                            await navigate({
                                to: '/admin/anticheat',
                                search: (prev) => ({ ...prev, server_id: info.getValue() as number })
                            });
                        }}
                    >
                        {info.row.original.server_name}
                    </Button>
                );
            }
        }),
        columnHelper.accessor('name', {
            header: 'Name',
            enableHiding: false,
            size: 300,
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.steam_id}
                    personaname={info.row.original.personaname}
                    avatar_hash={info.row.original.avatar}
                />
            )
        }),
        columnHelper.accessor('personaname', {
            enableHiding: true,
            header: 'Personaname'
        }),
        columnHelper.accessor('steam_id', {
            enableHiding: true,
            header: 'Steam ID'
        }),
        columnHelper.accessor('created_on', {
            header: 'Created',
            size: 140,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('demo_id', {
            header: 'Demo',
            size: 50,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('detection', {
            header: 'Detection',
            size: 130,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('triggered', {
            header: 'Count',
            size: 80,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('summary', {
            header: 'Summary',
            size: 400,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        })
    ];

    return (
        <Grid container spacing={2}>
            <Title>Anticheat Logs</Title>
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await form.handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'name'}
                                    children={(field) => {
                                        return <field.TextField label={'Name'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'steam_id'}
                                    children={(field) => {
                                        return <field.SteamIDField />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
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
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'detection'}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <>
                                                <FormControl fullWidth>
                                                    <InputLabel id="detection-select-label">Detection</InputLabel>
                                                    <Select
                                                        fullWidth
                                                        value={state.value}
                                                        label="Detection"
                                                        onChange={(e) => {
                                                            handleChange(e.target.value);
                                                        }}
                                                        onBlur={handleBlur}
                                                    >
                                                        <MenuItem value={0}>All</MenuItem>
                                                        {Detections.map((s) => (
                                                            <MenuItem value={s} key={s}>
                                                                {s}
                                                            </MenuItem>
                                                        ))}
                                                    </Select>
                                                </FormControl>
                                            </>
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'summary'}
                                    children={(field) => {
                                        return <field.TextField label={'Message'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppForm>
                                    <ButtonGroup>
                                        <form.ClearButton onClick={clear} />
                                        <form.ResetButton />
                                        <form.SubmitButton />
                                    </ButtonGroup>
                                </form.AppForm>
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeaderAndButtons title={`Entries`} iconLeft={<FilterAltIcon />}>
                    <FullTable<StacEntry>
                        // columnFilters={columnFilters}
                        pagination={pagination}
                        setPagination={setPagination}
                        data={logs ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        sorting={sorting}
                        columnVisibility={columnVisibility}
                        toOptions={{ from: Route.fullPath }}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
