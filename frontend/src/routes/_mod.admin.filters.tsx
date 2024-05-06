import { useState } from 'react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import TableCell from '@mui/material/TableCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetFilters, apiGetWarningState, Filter, FilterAction, filterActionString, UserWarning } from '../api/filters.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const filterSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['filter_id', 'is_regex', 'is_enabled', 'weight', 'trigger_count']).optional()
});

export const Route = createFileRoute('/_mod/admin/filters')({
    component: AdminFilters,
    validateSearch: (search) => filterSearchSchema.parse(search)
});

function AdminFilters() {
    const defaultRows = RowsPerPage.Ten;
    const { page, rows, sortOrder, sortColumn } = Route.useSearch();
    const { data: filters, isLoading } = useQuery({
        queryKey: ['filters', { page, rows, sortOrder, sortColumn }],
        queryFn: async () => {
            return await apiGetFilters({
                order_by: sortColumn ?? 'filter_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows)
            });
        }
    });

    const { data: warnings, isLoading: isLoadingWarnings } = useQuery({
        queryKey: ['filterWarnings'],
        queryFn: async () => {
            return await apiGetWarningState();
        }
    });
    // const onCreate = useCallback(async () => {
    //     try {
    //         const resp = await NiceModal.show<Filter>(ModalFilterEditor, {
    //             defaultPattern: '',
    //             defaultIsRegex: false
    //         });
    //         sendFlash('success', `Filter created successfully: ${resp.filter_id}`);
    //         setNewFilters((prevState) => {
    //             return [resp, ...prevState.filter((f) => f.filter_id != resp.filter_id)];
    //         });
    //     } catch (e) {
    //         sendFlash('error', `${e}`);
    //     }
    // }, [sendFlash]);
    //
    // const onEdit = useCallback(async (filter: Filter) => {
    //     try {
    //         const resp = await NiceModal.show<Filter>(ModalFilterEditor, {
    //             filter
    //         });
    //         setEditedFilters((prevState) => {
    //             return [...prevState, resp];
    //         });
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, []);
    //
    // const handleDelete = useCallback(
    //     async (filter: Filter) => {
    //         if (!filter.filter_id) {
    //             logErr(new Error('filter_id not present, cannot delete'));
    //             return;
    //         }
    //         apiDeleteFilter(filter.filter_id)
    //             .then(() => {
    //                 setDeletedFiltersIDs((prevState) => {
    //                     return [...prevState, filter.filter_id ?? 0];
    //                 });
    //                 sendFlash('success', `Deleted filter successfully`);
    //             })
    //             .catch((err) => {
    //                 sendFlash('error', `Failed to delete filter: ${err}`);
    //             });
    //     },
    //     [sendFlash]
    // );
    //
    // const onConfirmDelete = useCallback(
    //     async (filter: Filter) => {
    //         try {
    //             const confirmed = await NiceModal.show(ModalConfirm, {
    //                 title: 'Are you sure you want to delete this filter?'
    //             });
    //
    //             if (!confirmed) {
    //                 return;
    //             }
    //
    //             await handleDelete(filter);
    //         } catch (e) {
    //             logErr(e);
    //             return;
    //         }
    //     },
    //     [handleDelete]
    // );

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Word Filters'}
                    iconLeft={<FilterAltIcon />}
                    buttons={[
                        <ButtonGroup variant="contained" aria-label="outlined primary button group" key={`btn-headers-filters`}>
                            <Button startIcon={<AddBoxIcon />} color={'success'}>
                                New
                            </Button>
                        </ButtonGroup>
                    ]}
                >
                    <FiltersTable filters={filters ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} data={filters} path={'/admin/filters'} />
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
                        The way the warning tracking works is that each time a user triggers a match, it gets a entry in the table based on
                        the weight of the match. The individual match weight is determined by the word filter defined above. Once the sum of
                        their triggers exceeds the max weight the user will have action taken against them automatically. Matched entries
                        are ephemeral and are removed over time based on the configured timeout value.
                    </Typography>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const FiltersTable = ({ filters, isLoading }: { filters: LazyResult<Filter>; isLoading: boolean }) => {
    const columnHelper = createColumnHelper<Filter>();

    const columns = [
        columnHelper.accessor('pattern', {
            header: () => <TableHeadingCell name={'Pattern'} />,
            cell: (info) => <TableCellString>{`${info.getValue()}`}</TableCellString>
        }),
        columnHelper.accessor('is_regex', {
            header: () => <TableHeadingCell name={'Rx'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('action', {
            header: () => <TableHeadingCell name={'Action'} />,
            cell: (info) => <TableCellString>{filterActionString(info.getValue())}</TableCellString>
        }),
        columnHelper.accessor('duration', {
            header: () => <TableHeadingCell name={'Duration'} />,
            cell: (info) => (
                <TableCellString>{filters.data[info.row.index].action == FilterAction.Kick ? '' : info.getValue()}</TableCellString>
            )
        }),
        columnHelper.accessor('weight', {
            header: () => <TableHeadingCell name={'Weight'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        }),
        columnHelper.accessor('trigger_count', {
            header: () => <TableHeadingCell name={'Triggered'} />,
            cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
        })
    ];

    // const actions: ColumnDef<Filter> = {
    //     accessorFn: () => {
    //         return 'action';
    //     },
    //     header: () => <TableHeadingCell name={'Actions'} />,
    //     cell: () => (
    //         <ButtonGroup>
    //             <Tooltip title={'Edit filter'}>
    //                 <IconButton
    //                     color={'warning'}
    //                     onClick={async () => {
    //                         await onEdit(row);
    //                     }}
    //                 >
    //                     <EditIcon />
    //                 </IconButton>
    //             </Tooltip>
    //             <Tooltip title={'Delete filter'}>
    //                 <IconButton
    //                     color={'error'}
    //                     onClick={async () => {
    //                         await onConfirmDelete(row);
    //                     }}
    //                 >
    //                     <DeleteForeverIcon />
    //                 </IconButton>
    //             </Tooltip>
    //         </ButtonGroup>
    //     )
    // };

    const table = useReactTable({
        data: filters.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
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
