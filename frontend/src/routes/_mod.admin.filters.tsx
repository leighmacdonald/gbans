import { ChangeEvent, useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { z } from 'zod';
import { apiDeleteFilter, Filter, FilterAction, filterActionString } from '../api/filters.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { WarningStateContainer } from '../component/WarningStateContainer.tsx';
import { ModalConfirm, ModalFilterEditor } from '../component/modal';
import { LazyTable } from '../component/table/LazyTable.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { useWordFilters } from '../hooks/useWordFilters.ts';
import { logErr } from '../util/errors.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';

const filterSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['filter_id', 'is_regex', 'is_enabled', 'weight', 'trigger_count']).catch('filter_id')
});

export const Route = createFileRoute('/_mod/admin/filters')({
    component: AdminFilters,
    validateSearch: (search) => filterSearchSchema.parse(search)
});

function AdminFilters() {
    const [newFilters, setNewFilters] = useState<Filter[]>([]);
    const { page, rows, sortOrder, sortColumn } = Route.useSearch();
    const [deletedFiltersIDs, setDeletedFiltersIDs] = useState<number[]>([]);
    const [editedFilters, setEditedFilters] = useState<Filter[]>([]);
    const navigate = useNavigate();
    const { sendFlash } = useUserFlashCtx();

    const { data, count } = useWordFilters({
        order_by: sortColumn ?? 'filter_id',
        desc: (sortOrder ?? 'desc') == 'desc',
        limit: Number(rows ?? RowsPerPage.Ten),
        offset: Number((page ?? 0) * (rows ?? RowsPerPage.Ten))
    });

    const allRows = useMemo(() => {
        const edited = data.map((value) => {
            return editedFilters.find((f) => f.filter_id == value.filter_id) || value;
        });

        const undeleted = edited.filter((f) => f.filter_id && !deletedFiltersIDs.includes(f.filter_id));
        return [...newFilters, ...undeleted];
    }, [data, deletedFiltersIDs, editedFilters, newFilters]);

    const onCreate = useCallback(async () => {
        try {
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, {
                defaultPattern: '',
                defaultIsRegex: false
            });
            sendFlash('success', `Filter created successfully: ${resp.filter_id}`);
            setNewFilters((prevState) => {
                return [resp, ...prevState.filter((f) => f.filter_id != resp.filter_id)];
            });
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [sendFlash]);

    const onEdit = useCallback(async (filter: Filter) => {
        try {
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, {
                filter
            });
            setEditedFilters((prevState) => {
                return [...prevState, resp];
            });
        } catch (e) {
            logErr(e);
        }
    }, []);

    const handleDelete = useCallback(
        async (filter: Filter) => {
            if (!filter.filter_id) {
                logErr(new Error('filter_id not present, cannot delete'));
                return;
            }
            apiDeleteFilter(filter.filter_id)
                .then(() => {
                    setDeletedFiltersIDs((prevState) => {
                        return [...prevState, filter.filter_id ?? 0];
                    });
                    sendFlash('success', `Deleted filter successfully`);
                })
                .catch((err) => {
                    sendFlash('error', `Failed to delete filter: ${err}`);
                });
        },
        [sendFlash]
    );

    const onConfirmDelete = useCallback(
        async (filter: Filter) => {
            try {
                const confirmed = await NiceModal.show(ModalConfirm, {
                    title: 'Are you sure you want to delete this filter?'
                });

                if (!confirmed) {
                    return;
                }

                await handleDelete(filter);
            } catch (e) {
                logErr(e);
                return;
            }
        },
        [handleDelete]
    );

    return (
        <Stack spacing={2}>
            <ContainerWithHeaderAndButtons
                title={'Word Filters'}
                iconLeft={<FilterAltIcon />}
                buttons={[
                    <ButtonGroup variant="contained" aria-label="outlined primary button group" key={`btn-headers-filters`}>
                        <Button startIcon={<AddBoxIcon />} color={'success'} onClick={onCreate}>
                            New
                        </Button>
                    </ButtonGroup>
                ]}
            >
                <LazyTable<Filter>
                    showPager={true}
                    count={count}
                    rows={allRows}
                    page={Number(page ?? 0)}
                    rowsPerPage={Number(rows ?? RowsPerPage.Ten)}
                    sortOrder={sortOrder}
                    sortColumn={sortColumn}
                    onSortColumnChanged={async (column) => {
                        await navigate({ search: (prev) => ({ ...prev, sortColumn: column }) });
                    }}
                    onSortOrderChanged={async (direction) => {
                        await navigate({ search: (prev) => ({ ...prev, sortOrder: direction }) });
                    }}
                    onPageChange={async (_, newPage: number) => {
                        await navigate({ search: (prev) => ({ ...prev, page: newPage }) });
                    }}
                    onRowsPerPageChange={async (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                        await navigate({ search: (prev) => ({ ...prev, rows: Number(event.target.value), page: 0 }) });
                    }}
                    columns={[
                        {
                            label: 'Pattern',
                            tooltip: 'Pattern',
                            sortKey: 'pattern',
                            sortable: true,
                            align: 'left',
                            renderer: (row) => {
                                return row.pattern as string;
                            }
                        },
                        {
                            label: 'Regex',
                            tooltip: 'Regular Expression',
                            sortKey: 'is_regex',
                            sortable: true,
                            align: 'right',
                            renderer: (row) => {
                                return row.is_regex ? 'true' : 'false';
                            }
                        },
                        {
                            label: 'Action',
                            tooltip: 'What will happen when its triggered',
                            sortKey: 'action',
                            sortable: true,
                            align: 'right',
                            renderer: (row) => {
                                return filterActionString(row.action);
                            }
                        },
                        {
                            label: 'Duration',
                            tooltip: 'Duration when the action is triggered',
                            sortKey: 'duration',
                            sortable: false,
                            align: 'right',
                            renderer: (row) => {
                                return row.action == FilterAction.Kick ? '' : row.duration;
                            }
                        },
                        {
                            label: 'Weight',
                            tooltip: 'Weight per match',
                            sortKey: 'weight',
                            sortable: true,
                            align: 'right',
                            renderer: (_, weight) => {
                                return <Typography variant={'body2'}>{weight as number}</Typography>;
                            }
                        },
                        {
                            label: 'Triggered',
                            tooltip: 'Number of times the filter has been triggered',
                            sortKey: 'trigger_count',
                            sortable: true,
                            sortType: 'number',
                            align: 'right',
                            renderer: (row) => {
                                return row.trigger_count;
                            }
                        },
                        {
                            label: 'Actions',
                            tooltip: 'Action',
                            virtualKey: 'actions',
                            sortable: false,
                            align: 'right',
                            virtual: true,
                            renderer: (row) => {
                                return (
                                    <ButtonGroup>
                                        <Tooltip title={'Edit filter'}>
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    await onEdit(row);
                                                }}
                                            >
                                                <EditIcon />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={'Delete filter'}>
                                            <IconButton
                                                color={'error'}
                                                onClick={async () => {
                                                    await onConfirmDelete(row);
                                                }}
                                            >
                                                <DeleteForeverIcon />
                                            </IconButton>
                                        </Tooltip>
                                    </ButtonGroup>
                                );
                            }
                        }
                    ]}
                />
            </ContainerWithHeaderAndButtons>
            <WarningStateContainer />
        </Stack>
    );
}
