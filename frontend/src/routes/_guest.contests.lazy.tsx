import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Link from '@mui/material/Link';
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
import { apiContests, Contest } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

export const Route = createLazyFileRoute('/_guest/contests')({
    component: Contests
});

function Contests() {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: contests, isLoading } = useQuery({
        queryKey: ['contests'],
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
                        contests={contests ?? []}
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
    contests: Contest[];
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
                            // variant={'button'}
                            params={{ contest_id: info.row.original.contest_id }}
                        >
                            {info.getValue()}
                        </Link>
                    </TableCellSmall>
                );
            }
        }),
        columnHelper.accessor('num_entries', {
            header: () => <TableHeadingCell name={'Entries'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('date_start', {
            header: () => <TableHeadingCell name={'Stared On'} />,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('date_end', {
            header: () => <TableHeadingCell name={'Ends On'} />,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        })
    ];

    const table = useReactTable({
        data: contests,
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
