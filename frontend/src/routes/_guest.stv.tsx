import { useMemo, useState } from 'react';
import ClearIcon from '@mui/icons-material/Clear';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SearchIcon from '@mui/icons-material/Search';
import VideocamIcon from '@mui/icons-material/Videocam';
import { TablePagination } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Checkbox from '@mui/material/Checkbox';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, PermissionLevel, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';

const demosSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'created_on', 'map_name']).catch('created_on'),
    map_name: z.string().catch(''),
    server_id: z.number().catch(0),
    steam_id: z.string().catch(''),
    orderBy: z.enum(['map_name', 'created_on']).catch('created_on'),
    only_mine: z.boolean().catch(false)
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

type STVFilterForm = {
    server_id: number;
    steam_id: string;
    map_name: string;
    only_mine: boolean;
};

function STV() {
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
        queryKey: ['demos', { page, rows, map_name, steam_id, sortOrder, sortColumn, selectedSteamID, server_id }],
        queryFn: async () => {
            return await apiGetDemos({
                deleted: false,
                map_name: map_name,
                server_ids: server_id > 0 ? [server_id] : [],
                steam_id: selectedSteamID,
                offset: page * rows,
                limit: rows,
                desc: sortOrder == 'desc',
                order_by: sortColumn
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm<STVFilterForm>({
        onSubmit: async ({ value }) => {
            await navigate({ search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            map_name,
            server_id,
            steam_id,
            only_mine
        }
    });

    return (
        <Stack spacing={2}>
            <ContainerWithHeader title={'Filters'}>
                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        handleSubmit();
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
                                children={({ state, handleChange, handleBlur }) => {
                                    return (
                                        <>
                                            <TextField
                                                fullWidth
                                                label="Name"
                                                defaultValue={state.value}
                                                onChange={(e) => handleChange(e.target.value)}
                                                onBlur={handleBlur}
                                                variant="outlined"
                                            />
                                        </>
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={6} md={3}>
                            <Field
                                name={'steam_id'}
                                children={({ state, handleChange, handleBlur }) => {
                                    return (
                                        <>
                                            <TextField
                                                disabled={steamIdEnabled}
                                                fullWidth
                                                label="SteamID"
                                                value={state.value}
                                                onChange={(e) => handleChange(e.target.value)}
                                                onBlur={handleBlur}
                                                variant="outlined"
                                            />
                                        </>
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={2}>
                            <Field
                                name={'only_mine'}
                                children={({ state, handleChange, handleBlur }) => {
                                    return (
                                        <FormGroup>
                                            <FormControlLabel
                                                disabled={!hasPermission(PermissionLevel.User)}
                                                control={
                                                    <Checkbox
                                                        checked={state.value}
                                                        onBlur={handleBlur}
                                                        onChange={(_, v) => {
                                                            handleChange(v);
                                                            setSteamIdEnabled(v);
                                                        }}
                                                    />
                                                }
                                                label="Only Mine"
                                            />
                                        </FormGroup>
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => (
                                    <ButtonGroup>
                                        <Button
                                            type="submit"
                                            disabled={!canSubmit}
                                            variant={'contained'}
                                            color={'success'}
                                            startIcon={<SearchIcon />}
                                        >
                                            {isSubmitting ? '...' : 'Search'}
                                        </Button>
                                        <Button
                                            type="reset"
                                            onClick={() => reset()}
                                            variant={'contained'}
                                            color={'warning'}
                                            startIcon={<RestartAltIcon />}
                                        >
                                            Reset
                                        </Button>
                                        <Button
                                            type="button"
                                            onClick={async () => {
                                                await navigate({
                                                    search: (prev) => {
                                                        return {
                                                            ...prev,
                                                            page: 0,
                                                            steam_id: '',
                                                            body: '',
                                                            persona_name: '',
                                                            server_id: 0
                                                        };
                                                    }
                                                });
                                                // TODO fix this hackjob
                                                window.location.reload();
                                            }}
                                            variant={'contained'}
                                            color={'error'}
                                            startIcon={<ClearIcon />}
                                        >
                                            Clear
                                        </Button>
                                    </ButtonGroup>
                                )}
                            />
                        </Grid>
                    </Grid>
                </form>
            </ContainerWithHeader>
            <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
                <STVTable demos={demos ?? { data: [], count: 0 }} isLoading={isLoading} />
            </ContainerWithHeader>
        </Stack>
    );
}

const columnHelper = createColumnHelper<DemoFile>();

export const STVTable = ({ demos, isLoading }: { demos: LazyResult<DemoFile>; isLoading: boolean }) => {
    const { page, rows } = Route.useSearch();
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

    return (
        // <Formik<STVFormValues>
        //     onReset={onReset}
        //     onSubmit={onSubmit}
        //     validationSchema={validationSchema}
        //     validateOnChange={true}
        //     initialValues={{
        //         source_id: state.source,
        //         server_ids: state.serverIds ? state.serverIds : [],
        //         map_name: state.mapName,
        //         select_own: Boolean(state.own)
        //     }}
        // >
        <Grid container spacing={3}>
            <Grid xs={12}>
                <Grid container spacing={2}>
                    {/*<Grid md>*/}
                    {/*    <ServerIDsField />*/}
                    {/*</Grid>*/}
                    {/*<Grid md>*/}
                    {/*    <MapNameField />*/}
                    {/*</Grid>*/}
                    {/*<Grid md>*/}
                    {/*    <SourceIDField disabled={Boolean(state.own)} />*/}
                    {/*</Grid>*/}
                    {/*<Grid md>*/}
                    {/*    <SelectOwnField disabled={selectOwnDisabled} />*/}
                    {/*</Grid>*/}
                    {/*<Grid md>*/}
                    {/*    <FilterButtons />*/}
                    {/*</Grid>*/}
                </Grid>
            </Grid>
            <Grid xs={12}>
                <DataTable table={table} isLoading={isLoading} />
                <TablePagination
                    count={demos ? demos.count : 0}
                    page={page}
                    rowsPerPage={rows}
                    onPageChange={async (_, newPage: number) => {
                        await navigate({ search: (search) => ({ ...search, page: newPage }) });
                    }}
                />
                {/*<LazyTable*/}
                {/*    showPager={true}*/}
                {/*    count={count}*/}
                {/*    rows={data}*/}
                {/*    page={Number(state.page ?? 0)}*/}
                {/*    rowsPerPage={Number(state.rows ?? RowsPerPage.TwentyFive)}*/}
                {/*    sortOrder={state.sortOrder}*/}
                {/*    sortColumn={state.sortColumn}*/}
                {/*    onSortColumnChanged={async (column) => {*/}
                {/*        setState({ sortColumn: column });*/}
                {/*    }}*/}
                {/*    onSortOrderChanged={async (direction) => {*/}
                {/*        setState({ sortOrder: direction });*/}
                {/*    }}*/}
                {/*    onPageChange={(_, newPage: number) => {*/}
                {/*        setState({ page: newPage });*/}
                {/*    }}*/}
                {/*    onRowsPerPageChange={(event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {*/}
                {/*        setState({*/}
                {/*            rows: Number(event.target.value),*/}
                {/*            page: 0*/}
                {/*        });*/}
                {/*    }}*/}
                {/*    columns={[*/}
                {/*        {*/}
                {/*            tooltip: 'Server',*/}
                {/*            label: 'Server',*/}
                {/*            sortKey: 'server_name_short',*/}
                {/*            align: 'left',*/}
                {/*            width: '150px'*/}
                {/*        },*/}
                {/*        {*/}
                {/*            tooltip: 'Created On',*/}
                {/*            label: 'Created On',*/}
                {/*            sortKey: 'created_on',*/}
                {/*            align: 'left',*/}
                {/*            width: '150px',*/}
                {/*            hideSm: true,*/}
                {/*            renderer: (row) => {*/}
                {/*                return renderDateTime(row.created_on);*/}
                {/*            }*/}
                {/*        },*/}

                {/*        {*/}
                {/*            tooltip: 'Map',*/}
                {/*            label: 'Map',*/}
                {/*            sortKey: 'map_name',*/}
                {/*            align: 'left',*/}
                {/*            renderer: (row) => {*/}
                {/*                const re = /^workshop\/(.+?)\.ugc\d+$/;*/}
                {/*                const match = row.map_name.match(re);*/}
                {/*                if (!match) {*/}
                {/*                    return row.map_name;*/}
                {/*                }*/}
                {/*                return match[1];*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            tooltip: 'Size',*/}
                {/*            label: 'Size',*/}
                {/*            sortKey: 'size',*/}
                {/*            align: 'left',*/}
                {/*            width: '100px',*/}
                {/*            renderer: (obj) => {*/}
                {/*                return humanFileSize(obj.size);*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            tooltip: 'Total Downloads',*/}
                {/*            label: '#',*/}
                {/*            align: 'left',*/}
                {/*            sortKey: 'downloads',*/}
                {/*            width: '50px',*/}
                {/*            renderer: (row) => {*/}
                {/*                return <Typography variant={'body1'}>{row.downloads}</Typography>;*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            tooltip: 'Create Report From Demo',*/}
                {/*            label: 'RP',*/}
                {/*            virtual: true,*/}
                {/*            align: 'center',*/}
                {/*            virtualKey: 'report',*/}
                {/*            width: '40px',*/}
                {/*            renderer: (row) => {*/}
                {/*                return (*/}
                {/*                    <IconButton*/}
                {/*                        color={'error'}*/}
                {/*                        onClick={async () => {*/}
                {/*                            sessionStorage.setItem('demoName', row.title);*/}
                {/*                            await navigate({ to: '/report' });*/}
                {/*                        }}*/}
                {/*                    >*/}
                {/*                        <FlagIcon />*/}
                {/*                    </IconButton>*/}
                {/*                );*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            tooltip: 'Download',*/}
                {/*            label: 'DL',*/}
                {/*            virtual: true,*/}
                {/*            align: 'center',*/}
                {/*            virtualKey: 'download',*/}
                {/*            width: '40px',*/}
                {/*            renderer: (row) => {*/}
                {/*                return (*/}
                {/*                                    <IconButton
                                        component={Link}
                                        href={`${window.gbans.asset_url}/${window.gbans.bucket_demo}/${row.title}`}
                                        color={'primary'}
                                    >
                                        <FileDownloadIcon />
                                    </IconButton>*/}
                {/*                );*/}
                {/*            }*/}
                {/*        }*/}
                {/*    ]}*/}
                {/*/>*/}
            </Grid>
        </Grid>
        // </Formik>
    );
};
