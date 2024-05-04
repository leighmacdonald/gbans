import AddBoxIcon from '@mui/icons-material/AddBox';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetFilters, Filter, FilterAction, filterActionString } from '../api/filters.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { WarningStateContainer } from '../component/WarningStateContainer.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';

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
        <Stack spacing={2}>
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
            <WarningStateContainer />
        </Stack>
    );
}

const columnHelper = createColumnHelper<Filter>();

const FiltersTable = ({ filters, isLoading }: { filters: LazyResult<Filter>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('pattern', {
            header: () => <HeadingCell name={'Pattern'} />,
            cell: (info) => <Typography variant={'body1'}>{`${info.getValue()}`}</Typography>
        }),
        columnHelper.accessor('is_regex', {
            header: () => <HeadingCell name={'Rx'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('action', {
            header: () => <HeadingCell name={'Action'} />,
            cell: (info) => <Typography>{filterActionString(info.getValue())}</Typography>
        }),
        columnHelper.accessor('duration', {
            header: () => <HeadingCell name={'Duration'} />,
            cell: (info) => <Typography>{filters.data[info.row.index].action == FilterAction.Kick ? '' : info.getValue()}</Typography>
        }),
        columnHelper.accessor('weight', {
            header: () => <HeadingCell name={'Weight'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('trigger_count', {
            header: () => <HeadingCell name={'Triggered'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        })
    ];

    // const actions: ColumnDef<Filter> = {
    //     accessorFn: () => {
    //         return 'action';
    //     },
    //     header: () => <HeadingCell name={'Actions'} />,
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
