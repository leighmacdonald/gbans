import { ChangeEvent } from 'react';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import TimelineIcon from '@mui/icons-material/Timeline';
import { TablePagination } from '@mui/material';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetMatches, MatchSummary } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const matchSummarySchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['match_id', 'map_name']).catch('match_id'),
    map: z.string().catch('')
});

export const Route = createFileRoute('/_auth/logs/$steamId')({
    component: MatchListPage,
    validateSearch: (search) => matchSummarySchema.parse(search)
});

function MatchListPage() {
    const { sortColumn, map, sortOrder, page, rows } = Route.useSearch();
    const { steamId } = Route.useParams();

    const { data: matches, isLoading } = useQuery({
        queryKey: ['logs', { page, steamId, rows, sortOrder, sortColumn }],
        queryFn: async () => {
            return await apiGetMatches({
                steam_id: steamId,
                limit: Number(rows ?? RowsPerPage.Ten),
                offset: Number((page ?? 0) * (rows ?? RowsPerPage.Ten)),
                order_by: sortColumn ?? 'match_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                map: map
            });
        }
    });

    return (
        <Grid container>
            <Grid xs={12}>
                <MatchSummaryTable matches={matches?.data ?? []} count={matches?.count ?? 0} isLoading={isLoading} />
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<MatchSummary>();

const MatchSummaryTable = ({
    count,
    matches,
    isLoading
}: {
    matches: MatchSummary[];
    count: number;
    isLoading: boolean;
}) => {
    const { page, rows } = Route.useSearch();
    const navigate = useNavigate();

    const columns = [
        columnHelper.accessor('title', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => {
                return (
                    <Link
                        component={RouterLink}
                        variant={'button'}
                        to={'/match/$matchId'}
                        params={{ matchId: matches[info.row.index].match_id }}
                    >
                        {info.getValue()}
                    </Link>
                );
            }
        }),
        columnHelper.accessor('map_name', {
            header: () => <TableHeadingCell name={'Map'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('score_red', {
            header: () => <TableHeadingCell name={'RED'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('score_blu', {
            header: () => <TableHeadingCell name={'BLU'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('is_winner', {
            header: () => <TableHeadingCell name={'W'} />,
            cell: (info) => {
                return info.getValue() ? <CheckIcon color={'success'} /> : <CloseIcon color={'error'} />;
            }
        }),
        columnHelper.accessor('time_end', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: matches,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return (
        <Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'Match History'} iconLeft={<TimelineIcon />}>
                    <DataTable table={table} isLoading={isLoading} />
                </ContainerWithHeader>
            </Grid>
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
        </Grid>
    );
};
