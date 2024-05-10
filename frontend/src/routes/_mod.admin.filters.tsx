import { useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import CancelIcon from '@mui/icons-material/Cancel';
import EditIcon from '@mui/icons-material/Edit';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import TableCell from '@mui/material/TableCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import {
    ColumnDef,
    ColumnSort,
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    OnChangeFn,
    PaginationState,
    RowSelectionState,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { z } from 'zod';
import {
    apiDeleteFilter,
    apiGetFilters,
    apiGetWarningState,
    Filter,
    filterActionString,
    UserWarning
} from '../api/filters.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { IndeterminateCheckbox } from '../component/IndeterminateCheckbox.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { ModalConfirm, ModalFilterEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { findSelectedRow } from '../util/findSelectedRow.ts';
import { findSelectedRows } from '../util/findSelectedRows.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const filterSearchSchema = z.object({
    sortColumn: z.string().optional(),
    desc: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/filters')({
    component: AdminFilters,
    validateSearch: (search) => filterSearchSchema.parse(search)
});

function AdminFilters() {
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();
    const { sortColumn, desc } = Route.useSearch();
    const queryClient = useQueryClient();
    const [rowSelection, setRowSelection] = useState({});
    const [sorting, setSorting] = useState<SortingState>([
        {
            id: sortColumn ?? 'pattern',
            desc: desc ?? true
        }
    ]);

    const onSort = async (sortColumn: ColumnSort) => {
        console.log(sortColumn);
        await navigate({
            to: '/admin/filters',
            replace: true,
            search: { sortColumn: sortColumn.id, desc: sortColumn.desc }
        });
    };

    // const { page, rows, sortOrder, sortColumn } = Route.useSearch();
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: filters, isLoading } = useQuery({
        queryKey: ['filters'],
        queryFn: async () => {
            return await apiGetFilters();
        }
    });

    const { data: warnings, isLoading: isLoadingWarnings } = useQuery({
        queryKey: ['filterWarnings'],
        queryFn: async () => {
            return await apiGetWarningState();
        }
    });

    const onCreate = useCallback(async () => {
        try {
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, {});
            queryClient.setQueryData(['filters'], [...(filters ?? []), resp]);
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [filters, queryClient, sendFlash]);

    const onEdit = useCallback(async () => {
        try {
            const filter = findSelectedRow(rowSelection, filters ?? []);
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, { filter });

            queryClient.setQueryData(
                ['filters'],
                (filters ?? []).map((f) => {
                    return f.filter_id == resp.filter_id ? resp : f;
                })
            );
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [filters, queryClient, rowSelection, sendFlash]);

    const deleteMutation = useMutation({
        mutationKey: ['filters'],
        mutationFn: async (filter_id: number) => {
            await apiDeleteFilter(filter_id);
        },
        onSuccess: (_, filterId) => {
            sendFlash('error', `Deleted filter: ${filterId}`);
        }
    });

    const onDelete = useCallback(async () => {
        const selectedFiltersIds = findSelectedRows(rowSelection, filters ?? [])?.map((f) => f.filter_id);
        if (!selectedFiltersIds) {
            return;
        }

        try {
            const confirmed = await NiceModal.show(ModalConfirm, {
                title: `Are you sure you want to delete ${selectedFiltersIds.length} filter(s)?`
            });

            if (!confirmed) {
                return;
            }

            selectedFiltersIds.map((f) => {
                deleteMutation.mutate(f as number);
            });
            queryClient.setQueryData(
                ['filters'],
                (filters ?? []).filter((filter) => !selectedFiltersIds.includes(filter.filter_id))
            );
            setRowSelection({});
        } catch (e) {
            sendFlash('error', `${e}`);
            return;
        }
    }, [deleteMutation, filters, queryClient, rowSelection, sendFlash]);

    return (
        <Grid container spacing={2}>
            <Title>Filtered Words</Title>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={`Word Filters ${Object.values(rowSelection).length ? `Selected: ${Object.values(rowSelection).length}` : ''}`}
                    iconLeft={<FilterAltIcon />}
                    buttons={[
                        <ButtonGroup
                            variant="contained"
                            aria-label="outlined primary button group"
                            key={`btn-headers-filters`}
                        >
                            <Button
                                disabled={Object.values(rowSelection).length == 0}
                                color={'error'}
                                onClick={onDelete}
                                startIcon={<CancelIcon />}
                            >
                                Delete
                            </Button>
                            <Button
                                disabled={Object.values(rowSelection).length != 1}
                                color={'warning'}
                                onClick={onEdit}
                                startIcon={<EditIcon />}
                            >
                                Edit
                            </Button>
                            <Button startIcon={<AddBoxIcon />} color={'success'} onClick={onCreate}>
                                New
                            </Button>
                        </ButtonGroup>
                    ]}
                >
                    <FiltersTable
                        filters={filters ?? []}
                        isLoading={isLoading}
                        rowSelection={rowSelection}
                        setRowSelection={setRowSelection}
                        pagination={pagination}
                        setPagination={setPagination}
                        setSorting={setSorting}
                        sorting={sorting}
                        onSort={onSort}
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
                        count={filters?.length ?? 0}
                        rows={pagination.pageSize}
                        page={pagination.pageIndex}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={`Current Warning State (Max Weight: ${warnings?.max_weight ?? '...'})`}
                    iconLeft={<WarningIcon />}
                >
                    <WarningStateTable warnings={warnings?.current ?? []} isLoading={isLoadingWarnings} />
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'How it works'} iconLeft={<InfoIcon />}>
                    <Typography variant={'body1'}>
                        The way the warning tracking works is that each time a user triggers a match, it gets a entry in
                        the table based on the weight of the match. The individual match weight is determined by the
                        word filter defined above. Once the sum of their triggers exceeds the max weight the user will
                        have action taken against them automatically. Matched entries are ephemeral and are removed over
                        time based on the configured timeout value.
                    </Typography>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const FiltersTable = ({
    filters,
    isLoading,
    rowSelection,
    setRowSelection,
    pagination,
    setPagination,
    sorting,
    setSorting,
    onSort
}: {
    filters: Filter[];
    isLoading: boolean;
    rowSelection: RowSelectionState;
    setRowSelection: OnChangeFn<RowSelectionState>;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
    sorting: SortingState;
    setSorting: OnChangeFn<SortingState>;
    onSort: (sortColumn: ColumnSort) => void;
}) => {
    // const columnHelper = createColumnHelper<Filter>();
    const columns = useMemo<ColumnDef<Filter>[]>(
        () => [
            {
                id: 'select',
                header: ({ table }) => (
                    <IndeterminateCheckbox
                        {...{
                            checked: table.getIsAllRowsSelected(),
                            indeterminate: table.getIsSomeRowsSelected(),
                            onChange: table.getToggleAllRowsSelectedHandler()
                        }}
                    />
                ),
                cell: ({ row }) => (
                    <div className="px-1">
                        <IndeterminateCheckbox
                            {...{
                                checked: row.getIsSelected(),
                                disabled: !row.getCanSelect(),
                                indeterminate: row.getIsSomeSelected(),
                                onChange: row.getToggleSelectedHandler()
                            }}
                        />
                    </div>
                )
            },

            {
                accessorKey: 'pattern',
                cell: (info) => info.getValue(),
                enableSorting: true
            },
            {
                accessorKey: 'is_regex',
                accessorFn: (originalRow) => originalRow.is_regex,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />,
                header: () => <TableHeadingCell name={'Rx'} />,
                enableSorting: true
            },
            {
                accessorKey: 'action',
                accessorFn: (originalRow) => originalRow.action,
                cell: (info) => {
                    return (
                        <TableCellString>
                            {typeof filters[info.row.index] === 'undefined'
                                ? ''
                                : filterActionString(filters[info.row.index].action)}
                        </TableCellString>
                    );
                },
                header: () => <TableHeadingCell name={'Action'} />,
                enableSorting: true
            },
            {
                accessorKey: 'duration',
                accessorFn: (originalRow) => originalRow.duration,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: () => <TableHeadingCell name={'Duration'} />,
                enableSorting: true
            },
            {
                accessorKey: 'weight',
                accessorFn: (originalRow) => originalRow.weight,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: () => <TableHeadingCell name={'Weight'} />,
                enableSorting: true
            },
            {
                accessorKey: 'trigger_count',
                accessorFn: (originalRow) => originalRow.trigger_count,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: () => <TableHeadingCell name={'Enabled'} />,
                enableSorting: true
            }
        ],
        [filters]
    );

    const table = useReactTable({
        data: filters,
        columns: columns,
        autoResetPageIndex: true,
        enableRowSelection: true,
        enableSorting: true,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onRowSelectionChange: setRowSelection,
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        state: {
            rowSelection,
            pagination,
            sorting
        }
    });

    return <DataTable table={table} isLoading={isLoading} onSort={onSort} />;
};

export const WarningStateTable = ({ warnings, isLoading }: { warnings: UserWarning[]; isLoading: boolean }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const renderFilter = (f: Filter) => {
        const pat = f.is_regex ? (f.pattern as string) : (f.pattern as string);

        return (
            <>
                <Typography variant={'h6'}>Matched {f.is_regex ? 'Regex' : 'Text'}</Typography>
                <Typography variant={'body1'}>{pat}</Typography>
                <Typography variant={'body1'}>Weight: {f.weight}</Typography>
                <Typography variant={'body1'}>Action: {filterActionString(f.action)}</Typography>
            </>
        );
    };
    const columnHelper = createColumnHelper<UserWarning>();

    const columns = [
        columnHelper.accessor('steam_id', {
            header: () => <TableHeadingCell name={'Pattern'} />,
            cell: (info) => (
                <TableCellSmall>
                    <PersonCell
                        steam_id={info.getValue()}
                        personaname={warnings[info.row.index].personaname}
                        avatar_hash={warnings[info.row.index].avatar}
                    />
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Rx'} />,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('server_name', {
            header: () => <TableHeadingCell name={'Action'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('matched', {
            header: () => <TableHeadingCell name={'Duration'} />,
            cell: (info) => (
                <TableCell>
                    <Tooltip title={renderFilter(warnings[info.row.index].matched_filter)}>
                        <Typography>{info.getValue()}</Typography>
                    </Tooltip>
                </TableCell>
            )
        }),
        columnHelper.accessor('current_total', {
            header: () => <TableHeadingCell name={'Weight'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('message', {
            header: () => <TableHeadingCell name={'Triggered'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        })
    ];

    const table = useReactTable({
        data: warnings,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
        state: {
            pagination
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
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
                count={warnings.length}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
