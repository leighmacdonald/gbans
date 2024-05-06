import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createLazyFileRoute } from '@tanstack/react-router';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    useReactTable
} from '@tanstack/react-table';
import { format } from 'date-fns';
import { apiContests, Contest } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { LazyResult, RowsPerPage } from '../util/table.ts';

export const Route = createLazyFileRoute('/_guest/contests')({
    component: Contests
});

function Contests() {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: contests, isLoading } = useQuery({
        queryKey: ['adminContests'],
        queryFn: async () => {
            return await apiContests();
        }
    });

    // const onEnter = useCallback(async (contest_id: string) => {
    //     try {
    //         await NiceModal.show(ModalContestEntry, { contest_id });
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, []);

    return (
        <Grid container>
            <Grid xs={12}>
                <ContainerWithHeader title={'Contests'} iconLeft={<InsightsIcon />}>
                    <ContestsTable
                        contests={contests ?? { data: [], count: 0 }}
                        isLoading={isLoading}
                        pagination={pagination}
                        setPagination={setPagination}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<Contest>();

const ContestsTable = ({
    contests,
    isLoading,
    pagination,
    setPagination
}: {
    contests: LazyResult<Contest>;
    isLoading: boolean;
    pagination?: PaginationState;
    setPagination?: OnChangeFn<PaginationState>;
}) => {
    const columns = [
        columnHelper.accessor('title', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => {
                return (
                    <TableCellSmall>
                        <Link
                            component={RouterLink}
                            to={`/contests/$contest_id}`}
                            variant={'button'}
                            params={{ contest_id: contests.data[info.row.index].contest_id }}
                        >
                            {info.getValue()}
                        </Link>
                    </TableCellSmall>
                );
            }
        }),
        columnHelper.accessor('num_entries', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('date_start', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{format(info.getValue(), 'dd/MM/yy H:m')}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('date_end', {
            header: () => <TableHeadingCell name={'Message'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{format(info.getValue(), 'dd/MM/yy H:m')}</Typography>
                </TableCellSmall>
            )
        })
    ];

    const table = useReactTable({
        data: contests.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true,
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        state: { pagination }
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
