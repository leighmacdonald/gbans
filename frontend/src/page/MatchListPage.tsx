import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import TimelineIcon from '@mui/icons-material/Timeline';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { IconButton, TablePagination } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import React, { JSX, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { apiGetMatches, MatchesQueryOpts, MatchSummary } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Order, RowsPerPage } from '../component/DataTable';
import { LazyTable } from '../component/LazyTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { PageNotFound } from './PageNotFound';

interface MatchSummaryTableProps {
    steam_id: string;
}

const MatchSummaryTable = ({ steam_id }: MatchSummaryTableProps) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof MatchSummary>('match_id');
    const [rows, setRows] = useState<MatchSummary[]>([]);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const { sendFlash } = useUserFlashCtx();
    const { currentUser } = useCurrentUserCtx();
    const navigate = useNavigate();

    useEffect(() => {
        const abortController = new AbortController();
        const opts: MatchesQueryOpts = {
            steam_id: steam_id,
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc'
        };
        apiGetMatches(opts, abortController)
            .then((resp) => {
                setTotalRows(resp.count);
                setRows(resp.matches);
            })
            .catch((e) => {
                logErr(e);
            });
        return () => abortController.abort();
    }, [page, rowPerPageCount, sortColumn, sortOrder, steam_id]);

    if (currentUser.steam_id != steam_id) {
        sendFlash(
            'error',
            'Permission denied. Only your own match list is viewable'
        );
        navigate(`/logs/${currentUser.steam_id}`);
        return;
    }

    return (
        <Grid>
            <Grid xs={'auto'}>
                <TablePagination
                    component="div"
                    variant={'head'}
                    page={page}
                    count={totalRows}
                    showFirstButton
                    showLastButton
                    rowsPerPage={rowPerPageCount}
                    onRowsPerPageChange={(
                        event: React.ChangeEvent<
                            HTMLInputElement | HTMLTextAreaElement
                        >
                    ) => {
                        setRowPerPageCount(parseInt(event.target.value, 10));
                        setPage(0);
                    }}
                    onPageChange={(_, newPage) => {
                        setPage(newPage);
                    }}
                />
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Match History'}
                    iconLeft={<TimelineIcon />}
                >
                    <LazyTable
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        rows={rows}
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
                                            onClick={() => {
                                                navigate(
                                                    `/log/${row.match_id}`
                                                );
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
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.title}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Map Name',
                                tooltip: 'Map Name',
                                sortKey: 'map_name',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.map_name}
                                    </Typography>
                                )
                            },
                            {
                                label: 'RED',
                                tooltip: 'RED Score',
                                sortKey: 'score_red',
                                align: 'left',
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.score_red}
                                    </Typography>
                                )
                            },
                            {
                                label: 'BLU',
                                tooltip: 'BLU Score',
                                sortKey: 'score_blu',
                                align: 'left',
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.score_blu}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Won',
                                tooltip: 'Won',
                                sortKey: 'is_winner',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => {
                                    return row.is_winner ? (
                                        <CheckIcon color={'success'} />
                                    ) : (
                                        <CloseIcon color={'error'} />
                                    );
                                }
                            },
                            {
                                label: 'Date',
                                tooltip: 'Date',
                                sortKey: 'time_start',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.time_start.toLocaleString()}
                                    </Typography>
                                )
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};

export const MatchListPage = (): JSX.Element => {
    const { steam_id } = useParams();

    if (!steam_id) {
        return <PageNotFound error={'Invalid steam id'} />;
    }

    return (
        <Grid container>
            <Grid xs={12}>
                <MatchSummaryTable steam_id={steam_id} />
            </Grid>
        </Grid>
    );
};
