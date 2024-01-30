import React, { useCallback, useMemo, useState } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
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
import { Filter, FilterAction, filterActionString } from '../api/filters';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { WarningStateContainer } from '../component/WarningStateContainer';
import { ModalFilterDelete, ModalFilterEditor } from '../component/modal';
import { LazyTable, RowsPerPage } from '../component/table/LazyTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useWordFilters } from '../hooks/useWordFilters';

export const AdminFiltersPage = () => {
    const [newFilters, setNewFilters] = useState<Filter[]>([]);
    const [state, setState] = useUrlState({
        page: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const { sendFlash } = useUserFlashCtx();

    const { data, count } = useWordFilters({
        order_by: state.sortColumn ?? 'filter_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten))
    });

    const allRows = useMemo(() => {
        return [...newFilters, ...data];
    }, [data, newFilters]);

    const onCreate = useCallback(async () => {
        try {
            const resp = await NiceModal.show<Filter>(ModalFilterEditor, {
                defaultPattern: '',
                defaultIsRegex: false
            });
            sendFlash(
                'success',
                `Filter created successfully: ${resp.filter_id}`
            );
            setNewFilters((prevState) => {
                return [
                    resp,
                    ...prevState.filter((f) => f.filter_id != resp.filter_id)
                ];
            });
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    }, [sendFlash]);

    const onEdit = useCallback(async (filter: Filter) => {
        await NiceModal.show(ModalFilterEditor, {
            filter
        });
    }, []);

    const onDelete = useCallback(async (filter: Filter) => {
        await NiceModal.show(ModalFilterDelete, {
            record: filter
        });
    }, []);

    return (
        <Stack spacing={2}>
            <ContainerWithHeaderAndButtons
                title={'Word Filters'}
                iconLeft={<FilterAltIcon />}
                buttons={[
                    <ButtonGroup
                        variant="contained"
                        aria-label="outlined primary button group"
                        key={`btn-headers-filters`}
                    >
                        <Button
                            startIcon={<AddBoxIcon />}
                            color={'success'}
                            onClick={onCreate}
                        >
                            New
                        </Button>
                    </ButtonGroup>
                ]}
            >
                <LazyTable<Filter>
                    showPager={true}
                    count={count}
                    rows={allRows}
                    page={Number(state.page ?? 0)}
                    rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                    sortOrder={state.sortOrder}
                    sortColumn={state.sortColumn}
                    onSortColumnChanged={async (column) => {
                        setState({ sortColumn: column });
                    }}
                    onSortOrderChanged={async (direction) => {
                        setState({ sortOrder: direction });
                    }}
                    onPageChange={(_, newPage: number) => {
                        setState({ page: newPage });
                    }}
                    onRowsPerPageChange={(
                        event: React.ChangeEvent<
                            HTMLInputElement | HTMLTextAreaElement
                        >
                    ) => {
                        setState({
                            rows: Number(event.target.value),
                            page: 0
                        });
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
                                return row.action == FilterAction.Kick
                                    ? ''
                                    : row.duration;
                            }
                        },
                        {
                            label: 'Weight',
                            tooltip: 'Weight per match',
                            sortKey: 'weight',
                            sortable: true,
                            align: 'right',
                            renderer: (_, weight) => {
                                return (
                                    <Typography variant={'body2'}>
                                        {weight as number}
                                    </Typography>
                                );
                            }
                        },
                        {
                            label: 'Triggered',
                            tooltip:
                                'Number of times the filter has been triggered',
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
                                                    await onDelete(row);
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
};

export default AdminFiltersPage;
