import FilterAltIcon from '@mui/icons-material/FilterAlt';
import FilterListIcon from '@mui/icons-material/FilterList';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate } from '@tanstack/react-router';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    TableOptions,
    useReactTable
} from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetAnticheatLogs, apiGetServers, Detection, Detections, ServerSimple, StacEntry } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { stringToColour } from '../util/colours.ts';
import { makeCommonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';
import { makeValidateSteamIDCallback } from '../util/validator/makeValidateSteamIDCallback.ts';

const schema = z.object({
    ...makeCommonTableSearchSchema([
        'anticheat_id',
        'personaname',
        'summary',
        'detection',
        'steam_id',
        'created_on',
        'server_name'
    ]),
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

    const { data: logs, isLoading } = useQuery({
        queryKey: ['anticheat', search],
        queryFn: async () => {
            try {
                return await apiGetAnticheatLogs({
                    server_id: search.server_id ?? 0,
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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            //setColumnFilters(initColumnFilter(value));
            await navigate({ to: '/admin/anticheat', search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                steam_id: makeValidateSteamIDCallback(),
                summary: z.string(),
                detection: z.string(),
                server_id: z.number({ coerce: true })
            })
        },
        defaultValues: {
            summary: search.summary ?? '',
            detection: search.detection ?? '',
            steam_id: search.steam_id ?? '',
            server_id: search.server_id ?? 0
        }
    });

    const clear = async () => {
        //setColumnFilters([]);
        reset();
        await navigate({
            to: '/admin/anticheat',
            search: (prev) => ({
                ...prev,
                server_id: undefined,
                steam_id: undefined,
                detection: undefined,
                summary: undefined
            })
        });
    };

    return (
        <Grid container spacing={2}>
            <Title>Anticheat Logs</Title>
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
                            <Grid xs={6} md={4}>
                                <Field
                                    name={'steam_id'}
                                    children={(props) => {
                                        return <SteamIDField {...props} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
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
                            <Grid xs={6} md={4}>
                                <Field
                                    name={'detection'}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <>
                                                <FormControl fullWidth>
                                                    <InputLabel id="detection-select-label">Detection</InputLabel>
                                                    <Select
                                                        defaultValue={''}
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
                            <Grid xs={12}>
                                <Field
                                    name={'summary'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Message'} />;
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
                <ContainerWithHeaderAndButtons title={`Entries`} iconLeft={<FilterAltIcon />}>
                    <AnticheatTable logs={logs ?? []} isLoading={isLoading} />
                    <Paginator
                        page={search.pageIndex ?? 0}
                        rows={search.pageSize ?? defaultRows}
                        path={'/admin/anticheat'}
                        data={{ data: [], count: -1 }}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const AnticheatTable = ({
    logs,
    isLoading,
    manualPaging = true,
    pagination,
    setPagination
}: {
    logs: StacEntry[];
    isLoading: boolean;
    manualPaging?: boolean;
    pagination?: PaginationState;
    setPagination?: OnChangeFn<PaginationState>;
}) => {
    const navigate = useNavigate({ from: '/chatlogs' });
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

    const opts: TableOptions<StacEntry> = {
        data: logs,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: manualPaging,
        autoResetPageIndex: true,
        ...(manualPaging
            ? {}
            : {
                  manualPagination: false,
                  onPaginationChange: setPagination,
                  getPaginationRowModel: getPaginationRowModel(),
                  state: { pagination }
              })
    };

    const table = useReactTable(opts);

    return <DataTable table={table} isLoading={isLoading} />;
};
