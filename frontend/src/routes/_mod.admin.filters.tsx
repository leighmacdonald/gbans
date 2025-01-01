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
import { createFileRoute } from '@tanstack/react-router';
import {
    ColumnDef,
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    RowSelectionState,
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
import { Title } from '../component/Title';
import { ModalConfirm, ModalFilterEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { findSelectedRow } from '../util/findSelectedRow.ts';
import { findSelectedRows } from '../util/findSelectedRows.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

const filterSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['filter_id', 'is_regex', 'is_enabled', 'weight', 'trigger_count']).optional()
});

export const Route = createFileRoute('/_mod/admin/filters')({
    component: AdminFilters,
    validateSearch: (search) => filterSearchSchema.parse(search)
});

function AdminFilters() {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const [rowSelection, setRowSelection] = useState({});

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
    setPagination
}: {
    filters: Filter[];
    isLoading: boolean;
    rowSelection: RowSelectionState;
    setRowSelection: OnChangeFn<RowSelectionState>;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
}) => {
    // const columnHelper = createColumnHelper<Filter>();
    const columns = useMemo<ColumnDef<Filter>[]>(
        () => [
            {
                id: 'select',
                size: 30,
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
                size: 600,
                header: 'Pattern',
                cell: (info) => info.getValue()
            },
            {
                accessorKey: 'is_regex',
                size: 30,
                meta: { tooltip: 'Is this a regular expression' },
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />,
                header: 'Rx'
            },
            {
                accessorKey: 'action',
                size: 50,
                meta: { tooltip: 'What action to take' },
                cell: (info) => {
                    return (
                        <TableCellString>
                            {typeof filters[info.row.index] === 'undefined'
                                ? ''
                                : filterActionString(filters[info.row.index].action)}
                        </TableCellString>
                    );
                },
                header: 'Action'
            },
            {
                accessorKey: 'duration',
                size: 50,
                meta: { tooltip: 'Duration of the punishment when triggered' },
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: 'Duration'
            },
            {
                accessorKey: 'weight',
                size: 50,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: 'Weight'
            },
            {
                accessorKey: 'trigger_count',
                size: 40,
                meta: { tooltip: 'Number of times the filter has been triggered' },
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>,
                header: 'Trig #'
            }
        ],
        [filters]
    );

    const table = useReactTable({
        data: filters,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true,
        enableRowSelection: true,
        onRowSelectionChange: setRowSelection,
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        state: {
            rowSelection,
            pagination
        }
    });

    return <DataTable table={table} isLoading={isLoading} />;
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
            header: 'Pattern',
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
            header: 'Created',
            size: 100,
            cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('matched_filter.action', {
            header: 'Action',
            size: 100,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>
                        {typeof info.row.original.matched_filter.action === 'undefined'
                            ? ''
                            : filterActionString(info.row.original.matched_filter.action)}
                    </Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('matched', {
            header: 'Duration',
            size: 100,
            cell: (info) => (
                <TableCell>
                    <Tooltip title={renderFilter(warnings[info.row.index].matched_filter)}>
                        <Typography>{info.getValue()}</Typography>
                    </Tooltip>
                </TableCell>
            )
        }),
        columnHelper.accessor('current_total', {
            header: 'Weight',
            size: 30,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('message', {
            header: 'Triggered',
            size: 400,
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
