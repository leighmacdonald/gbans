import VideocamIcon from '@mui/icons-material/Videocam';
import { TablePagination } from '@mui/material';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import stc from 'string-to-color';
import { z } from 'zod';
import { apiGetDemos, apiGetServers, DemoFile, ServerSimple } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { humanFileSize, renderDateTime } from '../util/text.tsx';

const demosSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'created_on', 'map_name']).catch('created_on'),
    map_name: z.string().catch(''),
    servers_ids: z.number().array().catch([]),
    steam_id: z.string().catch(''),
    orderBy: z.enum(['map_name', 'created_on']).catch('created_on')
});

export const Route = createFileRoute('/_guest/stv')({
    component: STV,
    validateSearch: (search) => demosSchema.parse(search),
    loader: async ({ context }) => {
        return {
            servers: await context.queryClient.ensureQueryData({
                queryKey: ['serversSimple'],
                queryFn: apiGetServers
            })
        };
    }
});

function STV() {
    const { page, sortColumn, steam_id, map_name, servers_ids, sortOrder, rows } = Route.useSearch();
    const { servers } = useLoaderData({ from: '/_guest/stv' }) as { servers: ServerSimple[] };
    const { data: demos, isLoading } = useQuery({
        queryKey: ['demos', { page, rows, map_name, steam_id, sortOrder, sortColumn }],
        queryFn: async () => {
            return await apiGetDemos({
                deleted: false,
                map_name: map_name,
                server_ids: servers_ids,
                steam_id: steam_id,
                offset: page * rows,
                limit: rows,
                desc: sortOrder == 'desc',
                order_by: sortColumn
            });
        }
    });

    return (
        <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
            <STVTable demos={demos ?? { data: [], count: 0 }} servers={servers} isLoading={isLoading} />
        </ContainerWithHeader>
    );
}

const columnHelper = createColumnHelper<DemoFile>();

export const STVTable = ({ demos, servers, isLoading }: { demos: LazyResult<DemoFile>; servers: ServerSimple[]; isLoading: boolean }) => {
    const { page, rows } = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });

    const columns = [
        columnHelper.accessor('server_id', {
            header: () => <HeadingCell name={'Server'} />,
            cell: (info) => {
                const serv = servers.find((s) => (s.server_id = info.getValue())) || { server_name: 'unk-1' };
                return (
                    <Button
                        sx={{
                            color: stc(serv.server_name)
                        }}
                        onClick={async () => {
                            await navigate({ search: (prev) => ({ ...prev, server_id: info.getValue() }) });
                        }}
                    >
                        {serv?.server_name}
                    </Button>
                );
            },
            footer: () => <HeadingCell name={'Server'} />
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>,
            footer: () => <HeadingCell name={'Created'} />
        }),
        columnHelper.accessor('map_name', {
            header: () => <HeadingCell name={'Map Name'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>,
            footer: () => <HeadingCell name={'Name'} />
        }),
        columnHelper.accessor('size', {
            header: () => <HeadingCell name={'Size'} />,
            cell: (info) => <Typography>{humanFileSize(info.getValue())}</Typography>,
            footer: () => <HeadingCell name={'Size'} />
        }),
        columnHelper.accessor('downloads', {
            header: () => <HeadingCell name={'DL'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>,
            footer: () => <HeadingCell name={'DL'} />
        }),
        columnHelper.accessor('asset.asset_id', {
            header: () => <HeadingCell name={'Links'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>,
            footer: () => <HeadingCell name={'Links'} />
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
