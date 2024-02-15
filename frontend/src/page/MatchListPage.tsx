import { ChangeEvent, JSX } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import useUrlState from '@ahooksjs/use-url-state';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import TimelineIcon from '@mui/icons-material/Timeline';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { IconButton, TablePagination } from '@mui/material';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTable, RowsPerPage } from '../component/table/LazyTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useMatchHistory } from '../hooks/useMatchHistory';
import { PageNotFoundPage } from './PageNotFoundPage';

interface MatchSummaryTableProps {
    steam_id: string;
}

const MatchSummaryTable = ({ steam_id }: MatchSummaryTableProps) => {
    const [state, setState] = useUrlState({
        page: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined,
        map: undefined
    });

    const { sendFlash } = useUserFlashCtx();
    const { currentUser } = useCurrentUserCtx();
    const navigate = useNavigate();

    const { data: matches, count } = useMatchHistory({
        steam_id: currentUser.steam_id,
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'match_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        map: state.map
    });

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
                    page={Number(state.page ?? 0)}
                    count={count}
                    showFirstButton
                    showLastButton
                    rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                    onRowsPerPageChange={(
                        event: ChangeEvent<
                            HTMLInputElement | HTMLTextAreaElement
                        >
                    ) => {
                        setState({
                            rows: Number(event.target.value),
                            page: 0
                        });
                    }}
                    onPageChange={(_, newPage) => {
                        setState({ page: newPage });
                    }}
                />
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Match History'}
                    iconLeft={<TimelineIcon />}
                >
                    <LazyTable
                        sortOrder={state.sortOrder}
                        sortColumn={state.sortColumn}
                        onSortColumnChanged={async (column) => {
                            setState({ sortColumn: column });
                        }}
                        onSortOrderChanged={async (direction) => {
                            setState({ sortOrder: direction });
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
        return <PageNotFoundPage error={'Invalid steam id'} />;
    }

    return (
        <Grid container>
            <Grid xs={12}>
                <MatchSummaryTable steam_id={steam_id} />
            </Grid>
        </Grid>
    );
};

export default MatchListPage;
