import { useMemo, useState } from 'react';
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
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, SortingState } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { FullTable } from '../component/FullTable.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { initColumnFilter, initPagination, makeCommonTableSearchSchema } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const demosSchema = z.object({
    ...makeCommonTableSearchSchema(['demo_id', 'server_id', 'created_on', 'map_name']),
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

    const { data: demos, isLoading } = useQuery({
        queryKey: ['demos'],
        queryFn: apiGetDemos
    });

    const { Field, Subscribe, handleSubmit, reset, setFieldValue } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({ search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: demosSchema
        },
        defaultValues: {
            map_name: search.map_name ?? '',
            server_id: search.server_id ?? 0,
            stats: search.stats ?? ''
        }
    });

    const clear = async () => {
        setColumnFilters([]);
        reset();
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
                header: () => <TableHeadingCell name={'ID'} />,
                cell: (info) => <Typography>#{info.getValue()}</Typography>
            }),
            columnHelper.accessor('server_id', {
                filterFn: (row, _, filterValue) => {
                    return filterValue == 0 || row.original.server_id == filterValue;
                },
                enableSorting: true,
                enableColumnFilter: true,
                header: () => <TableHeadingCell name={'Server'} />,
                cell: (info) => {
                    return (
                        <Button
                            sx={{
                                color: stc(info.row.original.server_name_short)
                            }}
                            onClick={async () => {
                                await navigate({
                                    search: (prev) => ({ ...prev, server_id: info.row.original.server_id })
                                });
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
                enableColumnFilter: true,
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
                    return filterValue == '' || Object.keys(row.original.stats).includes(filterValue);
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
    }, [columnHelper, isAuthenticated, navigate]);

    return (
        <Grid container spacing={2}>
            <Title>SourceTV</Title>
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
                                        await navigate({ search: (prev) => ({ ...prev, stats: profile.steam_id }) });
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
                    <FullTable
                        columnFilters={columnFilters}
                        pagination={pagination}
                        setPagination={setPagination}
                        data={demos ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        sorting={sorting}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
