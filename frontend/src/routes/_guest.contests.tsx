import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
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
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title.tsx';
import { ensureFeatureEnabled } from '../util/features.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

export const Route = createFileRoute('/_guest/contests')({
    component: Contests,
    beforeLoad: () => {
        ensureFeatureEnabled('contests_enabled');
    }
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
            <Title>Contests</Title>
            <Grid size={{ xs: 12 }}>
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
            header: 'Title',
            size: 700,
            cell: (info) => {
                return (
                    <TableCellSmall>
                        <TextLink
                            to={`/contests/$contest_id`}
                            params={{ contest_id: info.row.original.contest_id as string }}
                        >
                            {info.getValue()}
                        </TextLink>
                    </TableCellSmall>
                );
            }
        }),
        columnHelper.accessor('num_entries', {
            header: 'Entries',
            size: 75,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('date_start', {
            header: 'Stared On',
            size: 140,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('date_end', {
            header: 'Ends On',
            size: 140,
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
