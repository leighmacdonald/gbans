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
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title.tsx';
import { ModalContestEditor } from '../component/modal';
import { logErr } from '../util/errors.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const contestsSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['contest_id', 'deleted']).catch('contest_id'),
    deleted: z.boolean().catch(false)
});

export const Route = createFileRoute('/_mod/admin/contests')({
    component: AdminContests,
    validateSearch: (search) => contestsSearchSchema.parse(search)
});

function AdminContests() {
    //const modal = useModal(ModalConfirm);
    // const theme = useTheme();
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
                header: () => <TableHeadingCell name={'Title'} />,
                cell: (info) => <TableCellString>{String(info.getValue())}</TableCellString>
            },
            {
                accessorKey: 'public',
                header: () => <TableHeadingCell name={'Public'} tooltip={'Is this visible to regular users'} />,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'hide_submissions',
                header: () => <TableHeadingCell name={'Hide Sub.'} tooltip={'Are submissions hidden from public'} />,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'voting',
                header: () => <TableHeadingCell name={'Voting'} tooltip={'Is voting enabled on submissions'} />,
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'down_votes',
                header: () => (
                    <TableHeadingCell
                        name={'Down Votes'}
                        tooltip={'Is down voting enabled. Required voting to be enabled'}
                    />
                ),
                cell: (info) => <TableCellBool enabled={Boolean(info.getValue())} />
            },
            {
                accessorKey: 'max_submissions',
                header: () => (
                    <TableHeadingCell name={'Max Subs.'} tooltip={'Max number of submissions a single user can make'} />
                ),
                cell: (info) => <TableCellString>{String(info.getValue())}</TableCellString>
            },
            {
                accessorKey: 'min_permission_level',
                header: () => (
                    <TableHeadingCell
                        name={'Min. Perms'}
                        tooltip={'Minimum permission level required to participate'}
                    />
                ),
                cell: (info) => (
                    <TableCellString>{permissionLevelString(info.getValue() as PermissionLevel)}</TableCellString>
                )
            },
            {
                accessorKey: 'date_start',
                header: () => <TableHeadingCell name={'Starts'} tooltip={'Start date'} />,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                accessorKey: 'date_end',
                header: () => <TableHeadingCell name={'Ends'} tooltip={'End date'} />,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                accessorKey: 'updated_on',
                header: () => <TableHeadingCell name={'Updated'} />,
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                id: 'actions',
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
