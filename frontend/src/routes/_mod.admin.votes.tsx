import FilterListIcon from '@mui/icons-material/FilterList';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { ReportWithAuthor } from '../api';
import { apiVotesQuery, VoteResult } from '../api/votes.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const votesSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['target_id', 'source_id', 'success', 'created_on']).catch('created_on'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    success: z.number().catch(-1)
});

export const Route = createFileRoute('/_mod/admin/votes')({
    component: AdminVotes,
    validateSearch: (search) => votesSearchSchema.parse(search)
});

function AdminVotes() {
    const { success, page, sortColumn, rows, sortOrder, source_id, target_id } = Route.useSearch();

    const { data: votes, isLoading } = useQuery({
        queryKey: ['votes', {}],
        queryFn: async () => {
            return apiVotesQuery({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn,
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id,
                success: success
            });
        }
    });
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
                <VotesTable votes={votes ?? { data: [], count: 0 }} isLoading={isLoading} />
            </ContainerWithHeaderAndButtons>
        </Stack>
        // </Formik>
    );
}

const columnHelper = createColumnHelper<VoteResult>();

const VotesTable = ({ votes, isLoading }: { votes: LazyResult<VoteResult>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('source_id', {
            header: () => <HeadingCell name={'Initiator'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={votes.data[info.row.index].source_id}
                    personaname={votes.data[info.row.index].source_name}
                    avatar_hash={votes.data[info.row.index].source_avatar_hash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: () => <HeadingCell name={'Subject'} />,
            cell: (info) => {
                return (
                    <PersonCell
                        steam_id={votes.data[info.row.index].target_id}
                        personaname={votes.data[info.row.index].target_name}
                        avatar_hash={votes.data[info.row.index].target_avatar_hash}
                    />
                );
            }
        }),
        columnHelper.accessor('success', {
            header: () => <HeadingCell name={'Success'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('server_name', {
            header: () => <HeadingCell name={'Server'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: votes.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
