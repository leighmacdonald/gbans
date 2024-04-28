import FilterListIcon from '@mui/icons-material/FilterList';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { commonTableSearchSchema } from '../util/table.ts';

const votesSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_asn_id', 'source_id', 'target_id', 'deleted', 'reason', 'as_num', 'valid_until']).catch('ban_asn_id'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    success: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/votes')({
    component: AdminVotes,
    validateSearch: (search) => votesSearchSchema.parse(search)
});

function AdminVotes() {
    //
    // const { data, count } = useVotes({
    //     order_by: state.sortColumn ?? 'vote_id',
    //     desc: (state.sortOrder ?? 'desc') == 'desc',
    //     limit: Number(state.rows ?? RowsPerPage.TwentyFive),
    //     offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
    //     source_id: state.source_id,
    //     target_id: state.target_id,
    //     success: state.success ?? -1
    // });
    //
    // const onReset = useCallback(
    //     async (_: VoteResultFilterValues, formikHelpers: FormikHelpers<VoteResultFilterValues>) => {
    //         setState({
    //             page: undefined,
    //             rows: undefined,
    //             sortOrder: undefined,
    //             sortColumn: undefined,
    //             source_id: undefined,
    //             target_id: undefined
    //         });
    //         await formikHelpers.setFieldValue('source_id', '');
    //         await formikHelpers.setFieldValue('target_id', '');
    //     },
    //     [setState]
    // );
    //
    // const onSubmit = useCallback(
    //     (values: VoteResultFilterValues) => {
    //         setState((prevState) => {
    //             return {
    //                 ...prevState,
    //                 source_id: values.source_id,
    //                 target_id: values.target_id
    //             };
    //         });
    //     },
    //     [setState]
    // );

    return (
        // <Formik<VoteResultFilterValues>
        //     initialValues={{
        //         source_id: '',
        //         target_id: ''
        //     }}
        //     onReset={onReset}
        //     onSubmit={onSubmit}
        // >
        <Stack spacing={2}>
            <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />}>
                <Grid container spacing={2}>
                    {/*<Grid xs={4} sm={4} md={4}>*/}
                    {/*    <SourceIDField />*/}
                    {/*</Grid>*/}
                    {/*<Grid xs={4} sm={4} md={4}>*/}
                    {/*    <TargetIDField />*/}
                    {/*</Grid>*/}
                    {/*<Grid xs={4} sm={4} md={4}>*/}
                    {/*    <FilterButtons />*/}
                    {/*</Grid>*/}
                </Grid>
            </ContainerWithHeader>
            <ContainerWithHeaderAndButtons title={'Vote History'} iconLeft={<HowToVoteIcon />}>
                {/*<LazyTable<VoteResult>*/}
                {/*    showPager={true}*/}
                {/*    count={count}*/}
                {/*    rows={data}*/}
                {/*    page={Number(state.page ?? 0)}*/}
                {/*    rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}*/}
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
                {/*            label: 'Source',*/}
                {/*            tooltip: 'Vote Initiator',*/}
                {/*            sortKey: 'source_id',*/}
                {/*            sortable: true,*/}
                {/*            align: 'left',*/}
                {/*            renderer: (row) => {*/}
                {/*                return (*/}
                {/*                    <SteamIDSelectField*/}
                {/*                        steam_id={row.source_id}*/}
                {/*                        personaname={row.source_name}*/}
                {/*                        avatarhash={row.source_avatar_hash}*/}
                {/*                        field_name={'source_id'}*/}
                {/*                    />*/}
                {/*                );*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            label: 'Target',*/}
                {/*            tooltip: 'Vote Target',*/}
                {/*            sortKey: 'target_id',*/}
                {/*            sortable: true,*/}
                {/*            align: 'left',*/}
                {/*            renderer: (row) => {*/}
                {/*                return (*/}
                {/*                    <SteamIDSelectField*/}
                {/*                        steam_id={row.target_id}*/}
                {/*                        personaname={row.target_name}*/}
                {/*                        avatarhash={row.target_avatar_hash}*/}
                {/*                        field_name={'target_id'}*/}
                {/*                    />*/}
                {/*                );*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            label: 'Success',*/}
                {/*            tooltip: 'Was the vote successful',*/}
                {/*            sortable: true,*/}
                {/*            sortKey: 'success',*/}
                {/*            align: 'right',*/}
                {/*            renderer: (row) => {*/}
                {/*                return <TableCellBool enabled={row.success} />;*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            label: 'Server',*/}
                {/*            tooltip: 'Server',*/}
                {/*            sortKey: 'server_id',*/}
                {/*            sortable: true,*/}
                {/*            align: 'right',*/}
                {/*            renderer: (row) => {*/}
                {/*                return row.server_name;*/}
                {/*            }*/}
                {/*        },*/}
                {/*        {*/}
                {/*            label: 'Created On',*/}
                {/*            tooltip: 'When the vote occurred',*/}
                {/*            sortKey: 'created_on',*/}
                {/*            sortable: true,*/}
                {/*            align: 'right',*/}
                {/*            renderer: (row) => {*/}
                {/*                return renderDateTime(row.created_on);*/}
                {/*            }*/}
                {/*        }*/}
                {/*    ]}*/}
                {/*/>*/}
            </ContainerWithHeaderAndButtons>
        </Stack>
        // </Formik>
    );
}
