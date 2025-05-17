import { useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import {
    ColumnDef,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    useReactTable
} from '@tanstack/react-table';
import { z } from 'zod';
import { apiContests, Contest, PermissionLevel, permissionLevelString } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { Title } from '../component/Title.tsx';
import { PaginatorLocal } from '../component/forum/PaginatorLocal.tsx';
import { ModalContestEditor } from '../component/modal';
import { DataTable } from '../component/table/DataTable.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { logErr } from '../util/errors.ts';
import { initPagination, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

const contestsSearchSchema = z.object({
    ...makeCommonTableSearchSchema(['contest_id', 'deleted']),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/contests')({
    component: AdminContests,
    validateSearch: (search) => contestsSearchSchema.parse(search)
});

function AdminContests() {
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));

    const { data: contests, isLoading } = useQuery({
        queryKey: ['adminContests'],
        queryFn: async () => {
            return await apiContests();
        }
    });

    const onEditContest = useCallback(async (contest?: Contest) => {
        try {
            await NiceModal.show<Contest>(ModalContestEditor, { contest });
        } catch (e) {
            logErr(e);
        }
    }, []);

    // const onDeleteContest = useCallback(
    //     async (contest_id: string) => {
    //         try {
    //             await apiContestDelete(contest_id);
    //             await modal.hide();
    //         } catch (e) {
    //             logErr(e);
    //             throw e;
    //         }
    //     },
    //     [modal]
    // );

    return (
        <ContainerWithHeaderAndButtons
            title={'User Submission Contests'}
            iconLeft={<EmojiEventsIcon />}
            buttons={[
                <Button
                    key={'add-button'}
                    startIcon={<AddIcon />}
                    variant={'contained'}
                    onClick={async () => {
                        await onEditContest();
                    }}
                    color={'success'}
                >
                    New Contest
                </Button>
            ]}
        >
            <Title>Contests</Title>
            <ContestTable
                contests={contests ?? []}
                isLoading={isLoading}
                onEdit={onEditContest}
                pagination={pagination}
                setPagination={setPagination}
            />
            <PaginatorLocal
                onRowsChange={(rows) => {
                    setPagination((prev) => {
                        return { ...prev, pageSize: rows };
                    });
                }}
                onPageChange={(page) => {
                    setPagination((prev) => {
                        return { ...prev, pageIndex: page };
                    });
                }}
                count={contests?.length ?? 0}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </ContainerWithHeaderAndButtons>
    );
}

const ContestTable = ({
    contests,
    isLoading,
    onEdit,
    pagination,
    setPagination
}: {
    contests: Contest[];
    isLoading: boolean;
    onEdit: (person: Contest) => Promise<void>;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
}) => {
    const columns = useMemo<ColumnDef<Contest>[]>(
        () => [
            {
                accessorKey: 'title',
                header: 'Title',
                size: 200,
                cell: (info) => <TableCellString>{String(info.getValue())}</TableCellString>
            },
            {
                accessorKey: 'public',
                meta: { tooltip: 'Is this visible to regular users' },
                header: 'Public',
                size: 30,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'hide_submissions',
                meta: { tooltip: 'Are submissions hidden from public' },
                header: 'Hide Sub.',
                size: 70,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'voting',
                meta: { tooltip: 'Is voting enabled on submissions' },
                header: 'Voting',
                size: 70,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'down_votes',
                meta: { tooltip: 'Is down voting enabled. Required voting to be enabled' },
                header: 'Down Votes',
                size: 110,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'max_submissions',
                meta: { tooltip: 'Max number of submissions a single user can make' },
                header: 'Max Subs.',
                size: 100,
                cell: (info) => <TableCellString>{String(info.getValue())}</TableCellString>
            },
            {
                accessorKey: 'min_permission_level',
                meta: { tooltip: 'Minimum permission level required to participate' },
                header: 'Min. Perms',
                size: 100,
                cell: (info) => (
                    <TableCellString>{permissionLevelString(info.getValue() as PermissionLevel)}</TableCellString>
                )
            },
            {
                accessorKey: 'date_start',
                meta: { tooltip: 'Start date' },
                header: 'Starts',
                size: 150,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                accessorKey: 'date_end',
                meta: { tooltip: 'End date' },
                header: 'Ends',
                size: 150,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                accessorKey: 'updated_on',
                header: 'Updated',
                size: 150,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                id: 'actions',
                size: 30,
                cell: (info) => {
                    return (
                        <IconButton color={'warning'} onClick={() => onEdit(info.row.original)}>
                            <EditIcon />
                        </IconButton>
                    );
                }
            }
        ],
        [onEdit]
    );
    const table = useReactTable({
        data: contests,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
        state: {
            pagination
        }
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
