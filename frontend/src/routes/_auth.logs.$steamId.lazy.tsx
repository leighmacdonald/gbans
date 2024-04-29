import { ChangeEvent } from 'react';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import TimelineIcon from '@mui/icons-material/Timeline';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { IconButton, TablePagination } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { z } from 'zod';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { LazyTable } from '../component/table/LazyTable.tsx';
import { useMatchHistory } from '../hooks/useMatchHistory.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';

export const Route = createFileRoute('/_auth/logs/$steamId')({
    component: MatchListPage,
    validateSearch: (search) => logsSearchSchema.parse(search)
});

interface MatchSummaryTableProps {
    steam_id: string;
}

const logsSearchSchema = z.object({
    ...commonTableSearchSchema,
    map: z.string().optional(),
    sortColumn: z.enum(['match_id', 'map_name']).catch('match_id')
});

function MatchListPage() {
    const { steamId } = Route.useParams();

    return (
        <Grid container>
            <Grid xs={12}>
                <MatchSummaryTable steam_id={steamId} />
            </Grid>
        </Grid>
    );
}

const MatchSummaryTable = ({ steam_id }: MatchSummaryTableProps) => {
    const { sortColumn, map, sortOrder, page, rows } = Route.useSearch();

    const navigate = useNavigate();

    const { data: matches, count } = useMatchHistory({
        steam_id: steam_id,
        limit: Number(rows ?? RowsPerPage.Ten),
        offset: Number((page ?? 0) * (rows ?? RowsPerPage.Ten)),
        order_by: sortColumn ?? 'match_id',
        desc: (sortOrder ?? 'desc') == 'desc',
        map: map
    });

    return (
        <Grid>
            <Grid xs={'auto'}>
                <TablePagination
                    component="div"
                    variant={'head'}
                    page={Number(page ?? 0)}
                    count={count}
                    showFirstButton
                    showLastButton
                    rowsPerPage={Number(rows ?? RowsPerPage.Ten)}
                    onRowsPerPageChange={async (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                        await navigate({ search: (prev) => ({ ...prev, rows: Number(event.target.value), page: 0 }) });
                    }}
                    onPageChange={async (_, newPage) => {
                        await navigate({ search: (prev) => ({ ...prev, page: newPage }) });
                    }}
                />
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'Match History'} iconLeft={<TimelineIcon />}>
                    <LazyTable
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            await navigate({ search: (prev) => ({ ...prev, sortColumn: column }) });
                        }}
                        onSortOrderChanged={async (direction) => {
                            await navigate({ search: (prev) => ({ ...prev, sortOrder: direction }) });
                        }}
                        rows={matches}
                        columns={[
                            {
                                label: '',
                                tooltip: 'View Match Details',
                                sortKey: 'match_id',
                                align: 'left',
                                width: 40,
                                renderer: (row) => (
                                    <Tooltip title={'View Match'}>
                                        <IconButton
                                            color={'primary'}
                                            onClick={async () => {
                                                await navigate({ to: `/log/${row.match_id}` });
                                            }}
                                        >
                                            <VisibilityIcon />
                                        </IconButton>
                                    </Tooltip>
                                )
                            },
                            {
                                label: 'Server Name',
                                tooltip: 'Server Name',
                                sortKey: 'title',
                                sortable: true,
                                align: 'left',
                                width: '50%',
                                renderer: (row) => <Typography variant={'button'}>{row.title}</Typography>
                            },
                            {
                                label: 'Map Name',
                                tooltip: 'Map Name',
                                sortKey: 'map_name',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => <Typography variant={'button'}>{row.map_name}</Typography>
                            },
                            {
                                label: 'RED',
                                tooltip: 'RED Score',
                                sortKey: 'score_red',
                                align: 'left',
                                renderer: (row) => <Typography variant={'button'}>{row.score_red}</Typography>
                            },
                            {
                                label: 'BLU',
                                tooltip: 'BLU Score',
                                sortKey: 'score_blu',
                                align: 'left',
                                renderer: (row) => <Typography variant={'button'}>{row.score_blu}</Typography>
                            },
                            {
                                label: 'Won',
                                tooltip: 'Won',
                                sortKey: 'is_winner',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => {
                                    return row.is_winner ? <CheckIcon color={'success'} /> : <CloseIcon color={'error'} />;
                                }
                            },
                            {
                                label: 'Date',
                                tooltip: 'Date',
                                sortKey: 'time_start',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => <Typography variant={'button'}>{row.time_start.toLocaleString()}</Typography>
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
