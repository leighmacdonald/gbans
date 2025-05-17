import { useMemo, useState } from 'react';
import { ChevronLeft, CloudDownload } from '@mui/icons-material';
import FilterListIcon from '@mui/icons-material/FilterList';
import FlagIcon from '@mui/icons-material/Flag';
import VideocamIcon from '@mui/icons-material/Videocam';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import Grid from '@mui/material/Grid';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, SortingState } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, ServerSimple } from '../api';
import { ButtonLink } from '../component/ButtonLink.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Title } from '../component/Title.tsx';
import { FullTable } from '../component/table/FullTable.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { stringToColour } from '../util/colours.ts';
import { ensureFeatureEnabled } from '../util/features.ts';
import { initColumnFilter, initPagination, makeCommonTableSearchSchema } from '../util/table.ts';
import { humanFileSize } from '../util/text.tsx';
import { renderDateTime } from '../util/time.ts';
import { makeValidateSteamIDCallback } from '../util/validator/makeValidateSteamIDCallback.ts';

const demosSchema = z.object({
    ...makeCommonTableSearchSchema(['demo_id', 'server_id', 'created_on', 'map_name']),
    map_name: z.string().optional(),
    server_id: z.number().optional(),
    stats: z.string().optional()
});

export const Route = createFileRoute('/_guest/stv')({
    component: STV,
    beforeLoad: () => {
        ensureFeatureEnabled('demos_enabled');
    },
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
    const search = Route.useSearch();
    const { servers } = useLoaderData({ from: '/_guest/stv' });
    const { profile, isAuthenticated } = useRouteContext({ from: '/_guest/stv' });
    const [pagination, setPagination] = useState(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'demo_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));
    const theme = useTheme();

    const { data: demos, isLoading } = useQuery({
        queryKey: ['demos'],
        queryFn: apiGetDemos
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({ search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChange: z.object({
                map_name: z.string(),
                server_id: z.number({ coerce: true }),
                stats: z.string()
            }),
            onChangeAsyncDebounceMs: 200,
            onChangeAsync: z.object({
                map_name: z.string(),
                server_id: z.number({ coerce: true }),
                stats: makeValidateSteamIDCallback()
            })
        },
        defaultValues: {
            map_name: search.map_name ?? '',
            server_id: search.server_id ?? 0,
            stats: search.stats ?? ''
        }
    });

    const clear = async () => {
        setColumnFilters([]);
        form.reset();
        await navigate({
            search: (prev) => ({
                ...prev,
                map_name: undefined,
                server_id: undefined,
                stats: undefined
            })
        });
    };

    const columnHelper = createColumnHelper<DemoFile>();

    const columns = useMemo(() => {
        return [
            columnHelper.accessor('demo_id', {
                header: 'ID',
                size: 40,
                cell: (info) => <Typography>#{info.getValue()}</Typography>
            }),
            columnHelper.accessor('server_id', {
                filterFn: (row, _, filterValue) => {
                    return filterValue == 0 || row.original.server_id == filterValue;
                },
                size: 75,
                enableSorting: true,
                enableColumnFilter: true,
                header: 'Server',
                cell: (info) => {
                    return (
                        <Button
                            sx={{
                                color: stringToColour(info.row.original.server_name_short, theme.palette.mode)
                            }}
                            onClick={async () => {
                                await navigate({
                                    search: (prev) => ({ ...prev, server_id: info.row.original.server_id })
                                });
                                await form.handleSubmit();
                            }}
                        >
                            {info.row.original.server_name_short}
                        </Button>
                    );
                }
            }),
            columnHelper.accessor('created_on', {
                header: 'Created',
                size: 140,
                cell: (info) => <Typography>{renderDateTime(info.getValue() as Date)}</Typography>
            }),
            columnHelper.accessor('map_name', {
                enableColumnFilter: true,
                header: 'Map Name',
                size: 450,
                cell: (info) => <Typography>{info.getValue() as string}</Typography>
            }),
            columnHelper.accessor('size', {
                header: 'Size',
                size: 60,
                cell: (info) => <Typography>{humanFileSize(info.getValue() as number)}</Typography>
            }),
            columnHelper.accessor('stats', {
                enableColumnFilter: true,
                filterFn: (row, _, filterValue) => {
                    return filterValue == '' || Object.keys(row.original.stats).includes(filterValue);
                },
                header: 'Players',
                size: 60,
                cell: (info) => <Typography>{Object.keys(Object(info.getValue())).length}</Typography>
            }),

            columnHelper.display({
                id: 'report',
                size: 60,
                cell: (info) => (
                    <ButtonLink
                        disabled={!isAuthenticated()}
                        color={'error'}
                        startIcon={<FlagIcon />}
                        variant={'contained'}
                        to={'/report'}
                        search={{ demo_id: info.row.original.demo_id }}
                    >
                        Report
                    </ButtonLink>
                )
            }),
            columnHelper.display({
                id: 'download',
                size: 60,
                cell: (info) => (
                    <Button
                        color={'success'}
                        component={Link}
                        variant={'contained'}
                        href={`/asset/${info.row.original.asset_id}`}
                        startIcon={<CloudDownload />}
                    >
                        Download
                    </Button>
                )
            })
        ];
    }, [columnHelper, form.handleSubmit, isAuthenticated, navigate, theme.palette.mode]);

    return (
        <Grid container spacing={2}>
            <Title>SourceTV</Title>
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
                                    name={'server_id'}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <>
                                                <FormControl fullWidth>
                                                    <InputLabel id="server-select-label">Servers</InputLabel>
                                                    <Select
                                                        fullWidth
                                                        value={state.value}
                                                        variant={'outlined'}
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
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'map_name'}
                                    children={(field) => {
                                        return <field.TextField label={'Map Name'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'stats'}
                                    children={(field) => {
                                        return <field.TextField label={'Steam ID'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6, md: 3 }} padding={2}>
                                <Button
                                    fullWidth
                                    disabled={!isAuthenticated()}
                                    startIcon={<ChevronLeft />}
                                    variant={'contained'}
                                    onClick={async () => {
                                        await navigate({ search: (prev) => ({ ...prev, stats: profile.steam_id }) });
                                        await form.handleSubmit();
                                    }}
                                >
                                    My SteamID
                                </Button>
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
                <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
                    <FullTable
                        columnFilters={columnFilters}
                        pagination={pagination}
                        setPagination={setPagination}
                        data={demos ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        sorting={sorting}
                        toOptions={{ to: Route.fullPath }}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
