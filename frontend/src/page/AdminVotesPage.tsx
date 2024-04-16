import { ChangeEvent } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Stack from '@mui/material/Stack';
import { VoteResult } from '../api/votes.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { WarningStateContainer } from '../component/WarningStateContainer';
import { LazyTable } from '../component/table/LazyTable';
import { useVotes } from '../hooks/useVotes.ts';
import { RowsPerPage } from '../util/table.ts';

export const AdminVotesPage = () => {
    const [state, setState] = useUrlState({
        page: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const { data, count } = useVotes({
        order_by: state.sortColumn ?? 'vote_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten))
    });

    return (
        <Stack spacing={2}>
            <ContainerWithHeaderAndButtons
                title={'Vote History'}
                iconLeft={<FilterAltIcon />}
            >
                <LazyTable<VoteResult>
                    showPager={true}
                    count={count}
                    rows={data}
                    page={Number(state.page ?? 0)}
                    rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                    sortOrder={state.sortOrder}
                    sortColumn={state.sortColumn}
                    onSortColumnChanged={async (column) => {
                        setState({ sortColumn: column });
                    }}
                    onSortOrderChanged={async (direction) => {
                        setState({ sortOrder: direction });
                    }}
                    onPageChange={(_, newPage: number) => {
                        setState({ page: newPage });
                    }}
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
                    columns={[
                        {
                            label: 'Source',
                            tooltip: 'Vote Initiatior',
                            sortKey: 'source_id',
                            sortable: true,
                            align: 'left',
                            renderer: (row) => {
                                return row.source_id;
                            }
                        },
                        {
                            label: 'Target',
                            tooltip: 'Vote Target',
                            sortKey: 'target_id',
                            sortable: true,
                            align: 'left',
                            renderer: (row) => {
                                return row.target_id;
                            }
                        },
                        {
                            label: 'Success',
                            tooltip: 'Was the vote successful',
                            sortKey: 'success',
                            sortable: true,
                            align: 'right',
                            renderer: (row) => {
                                return String(row.success);
                            }
                        },
                        {
                            label: 'Server',
                            tooltip: 'Server',
                            sortKey: 'server_id',
                            sortable: true,
                            align: 'right',
                            renderer: (row) => {
                                return row.server_id;
                            }
                        },
                        {
                            label: 'Created On',
                            tooltip: 'When the vote occurred',
                            sortKey: 'created_on',
                            sortable: false,
                            align: 'right',
                            renderer: (row) => {
                                return row.created_on.toString();
                            }
                        }
                    ]}
                />
            </ContainerWithHeaderAndButtons>
            <WarningStateContainer />
        </Stack>
    );
};

export default AdminVotesPage;
