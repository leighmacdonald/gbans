import { useMemo } from 'react';
import HistoryIcon from '@mui/icons-material/History';
import InfoIcon from '@mui/icons-material/Info';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { TablePagination } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, Link as RouterLink, useNavigate, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetReports, ReportStatus, reportStatusString, ReportWithAuthor } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { ReportCreateForm } from '../component/ReportCreateForm.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';

const reportLogsSchema = z.object({
    ...commonTableSearchSchema,
    rows: z.number().catch(RowsPerPage.Ten),
    sortColumn: z.enum(['report_status', 'created_on']).catch('created_on'),
    report_status: z.nativeEnum(ReportStatus).catch(ReportStatus.Any)
});

export const Route = createFileRoute('/_auth/report/')({
    component: ReportCreate,
    validateSearch: (search) => reportLogsSchema.parse(search)
});

function ReportCreate() {
    const { profile, userSteamID } = useRouteContext({ from: '/_auth/report/' });
    const { page, sortColumn, rows, sortOrder } = Route.useSearch();
    const navigate = useNavigate();

    const canReport = useMemo(() => {
        const user = profile();
        return user.steam_id && user.ban_id == 0;
    }, [profile]);

    const { data: logs, isLoading } = useQuery({
        queryKey: ['history', { page, userSteamID }],
        queryFn: async () => {
            return await apiGetReports({
                source_id: userSteamID,
                limit: Number(rows),
                offset: Number(page ?? 0) * Number(rows),
                order_by: sortColumn ?? 'created_on',
                desc: sortOrder == 'desc',
                report_status: ReportStatus.Any
            });
        }
    });

    return (
        <Grid container spacing={3}>
            <Grid xs={12} md={8}>
                <Stack spacing={2}>
                    {canReport && <ReportCreateForm />}
                    {!canReport && (
                        <ContainerWithHeader title={'Permission Denied'}>
                            <Typography variant={'body1'} padding={2}>
                                You are unable to report players while you are currently banned/muted.
                            </Typography>
                            <ButtonGroup sx={{ padding: 2 }}>
                                <Button component={RouterLink} variant={'contained'} color={'primary'} to={`/ban/${profile().ban_id}`}>
                                    Appeal Ban
                                </Button>
                            </ButtonGroup>
                        </ContainerWithHeader>
                    )}
                    <ContainerWithHeader title={'Your Report History'} iconLeft={<HistoryIcon />}>
                        {isLoading ? <LoadingPlaceholder /> : <UserReportHistory history={logs ?? { data: [], count: 0 }} />}
                        <TablePagination
                            count={logs ? logs.count : 0}
                            page={page}
                            rowsPerPage={rows}
                            onPageChange={async (_, newPage: number) => {
                                await navigate({ search: (search) => ({ ...search, page: newPage }) });
                            }}
                        />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
            <Grid xs={12} md={4}>
                <ContainerWithHeader title={'Reporting Guide'} iconLeft={<InfoIcon />}>
                    <List>
                        <ListItem>
                            <ListItemText>
                                Once your report is posted, it will be reviewed by a moderator. If further details are required you will be
                                notified about it.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                If you wish to link to a specific SourceTV recording, you can find them listed{' '}
                                <Link component={RouterLink} to={'/stv'}>
                                    here
                                </Link>
                                . Once you find the recording you want, you may select the report icon which will open a new report with the
                                demo attached. From there you will optionally be able to enter a specific tick if you have one.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                Reports that are made in bad faith, or otherwise are considered to be trolling will be closed, and the
                                reporter will be banned.
                            </ListItemText>
                        </ListItem>

                        <ListItem>
                            <ListItemText>
                                Its only possible to open a single report against a particular player. If you wish to add more evidence or
                                discuss further an existing report, please open the existing report and add it by creating a new message
                                there. You can see your current report history below.
                            </ListItemText>
                        </ListItem>
                    </List>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ReportWithAuthor>();

const UserReportHistory = ({ history }: { history: LazyResult<ReportWithAuthor> }) => {
    const columns = [
        columnHelper.accessor('report_status', {
            header: () => <HeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <Stack direction={'row'} spacing={1}>
                        <ReportStatusIcon reportStatus={info.getValue()} />
                        <Typography variant={'body1'}>{reportStatusString(info.getValue())}</Typography>
                    </Stack>
                );
            },
            footer: () => <HeadingCell name={'Server'} />
        }),
        columnHelper.accessor('subject', {
            header: () => <HeadingCell name={'Player'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={history.data[info.row.index].subject.steam_id}
                    personaname={history.data[info.row.index].subject.personaname}
                    avatar_hash={history.data[info.row.index].subject.avatarhash}
                />
            ),
            footer: () => <HeadingCell name={'Created'} />
        }),
        columnHelper.accessor('report_id', {
            header: () => <HeadingCell name={'View'} />,
            cell: (info) => (
                <ButtonGroup>
                    <IconButton color={'primary'} component={RouterLink} to={`/report/$reportId`} params={{ reportId: info.getValue() }}>
                        <Tooltip title={'View'}>
                            <VisibilityIcon />
                        </Tooltip>
                    </IconButton>
                </ButtonGroup>
            ),
            footer: () => <HeadingCell name={'Name'} />
        })
    ];

    const table = useReactTable({
        data: history.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return (
        <Stack>
            <DataTable table={table} />
        </Stack>
    );
};
