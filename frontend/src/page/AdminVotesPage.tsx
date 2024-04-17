import { ChangeEvent, useCallback } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import { VoteResult } from '../api/votes.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { FilterButtons } from '../component/formik/FilterButtons.tsx';
import { SourceIDField } from '../component/formik/SourceIDField.tsx';
import { SteamIDSelectField } from '../component/formik/SteamIDSelectField.tsx';
import { TargetIDField } from '../component/formik/TargetIdField.tsx';
import { LazyTable } from '../component/table/LazyTable';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { useVotes } from '../hooks/useVotes.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

type VoteResultFilterValues = {
    source_id: string;
    target_id: string;
};

export const AdminVotesPage = () => {
    const [state, setState] = useUrlState({
        page: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined,
        source_id: undefined,
        target_id: undefined,
        success: undefined
    });

    const { data, count } = useVotes({
        order_by: state.sortColumn ?? 'vote_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        source_id: state.source_id,
        target_id: state.target_id,
        success: state.success ?? -1
    });

    const onReset = useCallback(
        async (
            _: VoteResultFilterValues,
            formikHelpers: FormikHelpers<VoteResultFilterValues>
        ) => {
            setState({
                page: undefined,
                rows: undefined,
                sortOrder: undefined,
                sortColumn: undefined,
                source_id: undefined,
                target_id: undefined
            });
            await formikHelpers.setFieldValue('source_id', '');
            await formikHelpers.setFieldValue('target_id', '');
        },
        [setState]
    );

    const onSubmit = useCallback(
        (values: VoteResultFilterValues) => {
            setState((prevState) => {
                return {
                    ...prevState,
                    source_id: values.source_id,
                    target_id: values.target_id
                };
            });
        },
        [setState]
    );

    return (
        <Formik<VoteResultFilterValues>
            initialValues={{
                source_id: '',
                target_id: ''
            }}
            onReset={onReset}
            onSubmit={onSubmit}
        >
            <Stack spacing={2}>
                <Grid container spacing={2}>
                    <Grid xs={4} sm={4} md={4}>
                        <SourceIDField />
                    </Grid>
                    <Grid xs={4} sm={4} md={4}>
                        <TargetIDField />
                    </Grid>
                    <Grid xs={4} sm={4} md={4}>
                        <FilterButtons />
                    </Grid>
                </Grid>

                <ContainerWithHeaderAndButtons
                    title={'Vote History'}
                    iconLeft={<HowToVoteIcon />}
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
                                    return (
                                        <SteamIDSelectField
                                            steam_id={row.source_id}
                                            personaname={row.source_name}
                                            avatarhash={row.source_avatar_hash}
                                            field_name={'source_id'}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'Target',
                                tooltip: 'Vote Target',
                                sortKey: 'target_id',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => {
                                    return (
                                        <SteamIDSelectField
                                            steam_id={row.target_id}
                                            personaname={row.target_name}
                                            avatarhash={row.target_avatar_hash}
                                            field_name={'target_id'}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'Success',
                                tooltip: 'Was the vote successful',
                                sortable: true,
                                sortKey: 'success',
                                align: 'right',
                                renderer: (row) => {
                                    return (
                                        <TableCellBool enabled={row.success} />
                                    );
                                }
                            },
                            {
                                label: 'Server',
                                tooltip: 'Server',
                                sortKey: 'server_id',
                                sortable: true,
                                align: 'right',
                                renderer: (row) => {
                                    return row.server_name;
                                }
                            },
                            {
                                label: 'Created On',
                                tooltip: 'When the vote occurred',
                                sortKey: 'created_on',
                                sortable: false,
                                align: 'right',
                                renderer: (row) => {
                                    return renderDateTime(row.created_on);
                                }
                            }
                        ]}
                    />
                </ContainerWithHeaderAndButtons>
            </Stack>
        </Formik>
    );
};

export default AdminVotesPage;
