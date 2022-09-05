import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import minimatch from 'minimatch';
import { Heading } from '../component/Heading';
import SaveIcon from '@mui/icons-material/Save';
import TextField from '@mui/material/TextField';
import ButtonGroup from '@mui/material/ButtonGroup';
import EditIcon from '@mui/icons-material/Edit';
import AddIcon from '@mui/icons-material/Add';
import RemoveIcon from '@mui/icons-material/Remove';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import Button from '@mui/material/Button';
import { DataTable, RowsPerPage } from '../component/DataTable';
import { apiGetFilters, apiSaveFilter, Filter } from '../api/filters';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { ConfirmDeleteFilterModal } from '../component/ConfirmDeleteFilterModal';
import { Nullable } from '../util/types';

export const AdminFilters = () => {
    const [editFilter, setEditFilter] = useState<Filter>({
        word_id: 0,
        filter_name: '',
        patterns: []
    });

    const [newPattern, setNewPattern] = useState('');
    const [filters, setFilters] = useState<Filter[]>([]);
    const [deleteTarget, setDeleteTarget] = useState<Nullable<Filter>>();
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);

    const { sendFlash } = useUserFlashCtx();

    const reset = () => {
        apiGetFilters().then((resp) => {
            if (!resp.status) {
                return;
            }
            setFilters(resp.result || []);
        });
    };

    useEffect(() => {
        reset();
    }, []);

    const onSaveFilter = useCallback(() => {
        apiSaveFilter(editFilter).then((resp) => {
            if (!resp.status) {
                return;
            }
            sendFlash('success', 'Filter saved');
            setEditFilter({
                word_id: 0,
                filter_name: '',
                patterns: []
            });
            setNewPattern('');
            reset();
        });
    }, [editFilter, sendFlash]);

    return (
        <>
            {deleteTarget && (
                <ConfirmDeleteFilterModal
                    record={deleteTarget}
                    open={deleteModalOpen}
                    setOpen={setDeleteModalOpen}
                    onSuccess={() => {
                        setFilters(
                            filters.filter(
                                (f) => f.word_id != deleteTarget?.word_id
                            )
                        );
                        setDeleteModalOpen(false);
                    }}
                />
            )}
            <Grid container spacing={2} paddingTop={3}>
                <Grid item xs={4}>
                    <Paper elevation={1}>
                        <Heading>Filter Editor</Heading>
                        <Stack spacing={2} padding={2}>
                            <TextField
                                variant={'standard'}
                                margin={'dense'}
                                fullWidth
                                label={'Filter Name'}
                                value={editFilter.filter_name}
                                onChange={(evt) => {
                                    setEditFilter((f) => {
                                        return {
                                            ...f,
                                            filter_name: evt.target.value
                                        };
                                    });
                                }}
                            />
                            <Stack direction={'row'} spacing={2}>
                                <TextField
                                    key={`pattern-new`}
                                    margin={'dense'}
                                    variant={'standard'}
                                    fullWidth
                                    placeholder={'fric*'}
                                    value={newPattern}
                                    onChange={(evt) => {
                                        setNewPattern(evt.target.value);
                                    }}
                                />
                                <Button
                                    variant={'contained'}
                                    color={'success'}
                                    startIcon={<AddIcon />}
                                    onClick={(evt) => {
                                        evt.preventDefault();
                                        if (!newPattern) {
                                            return;
                                        }
                                        setEditFilter((f) => {
                                            return {
                                                ...f,
                                                patterns: [
                                                    ...f.patterns,
                                                    newPattern
                                                ]
                                            };
                                        });
                                        setNewPattern('');
                                    }}
                                >
                                    Add
                                </Button>
                            </Stack>
                            {editFilter.patterns.map((_, i) => {
                                return (
                                    <Stack
                                        direction={'row'}
                                        spacing={2}
                                        key={`pattern-edit-${i}`}
                                    >
                                        <TextField
                                            margin={'dense'}
                                            key={`pattern-${i}`}
                                            variant={'standard'}
                                            fullWidth
                                            value={editFilter.patterns[i]}
                                            onChange={(evt) => {
                                                setEditFilter((f) => {
                                                    const p =
                                                        editFilter.patterns;
                                                    p[i] = evt.target.value;
                                                    return {
                                                        ...f,
                                                        patterns: p
                                                    };
                                                });
                                            }}
                                        />
                                        <Button
                                            variant={'contained'}
                                            color={'error'}
                                            startIcon={<RemoveIcon />}
                                            onClick={() => {
                                                const p = editFilter.patterns;
                                                p.splice(i, 1);
                                                setEditFilter((f) => {
                                                    return {
                                                        ...f,
                                                        patterns: p
                                                    };
                                                });
                                                setNewPattern('');
                                            }}
                                        >
                                            Del
                                        </Button>
                                    </Stack>
                                );
                            })}
                            <ButtonGroup>
                                <Button
                                    color={'success'}
                                    variant={'contained'}
                                    startIcon={<SaveIcon />}
                                    onClick={onSaveFilter}
                                    disabled={
                                        !editFilter.filter_name ||
                                        editFilter.patterns.length == 0
                                    }
                                >
                                    Save Filter Rule
                                </Button>
                                <Button
                                    color={'warning'}
                                    variant={'contained'}
                                    startIcon={<SaveIcon />}
                                    onClick={() => {
                                        setEditFilter({
                                            word_id: 0,
                                            filter_name: '',
                                            patterns: []
                                        });
                                        setNewPattern('');
                                    }}
                                >
                                    Clear
                                </Button>
                            </ButtonGroup>
                        </Stack>
                    </Paper>
                </Grid>
                <Grid item xs={8}>
                    <Paper elevation={1}>
                        <Heading>Filters (Use filter to test matches)</Heading>
                        <DataTable
                            filterFn={(query, rows) => {
                                if (!query) {
                                    return rows;
                                }
                                return rows.filter((row) => {
                                    return (
                                        row.patterns.filter((pattern) =>
                                            minimatch(query, pattern)
                                        ).length > 0
                                    );
                                });
                            }}
                            columns={[
                                {
                                    label: '#',
                                    tooltip: 'Filter ID',
                                    sortKey: 'word_id',
                                    sortable: true,
                                    align: 'left',
                                    queryValue: (o) => `${o.word_id}`
                                },
                                {
                                    label: 'Name',
                                    tooltip: 'Filter Name',
                                    sortKey: 'filter_name',
                                    sortable: true,
                                    align: 'left',
                                    //width: '250px',
                                    queryValue: (o) => `${o.word_id}`
                                },
                                {
                                    label: 'Patterns',
                                    tooltip: 'Patterns',
                                    sortKey: 'patterns',
                                    sortable: false,
                                    align: 'left',
                                    //width: '100%',
                                    queryValue: (o) => `${o.word_id}`,
                                    renderer: (row) => {
                                        return row.patterns.join(', ');
                                    }
                                },
                                {
                                    label: 'Actions',
                                    tooltip: 'Action',
                                    virtualKey: 'actions',
                                    sortable: false,
                                    align: 'right',
                                    virtual: true,
                                    queryValue: (o) => `${o.word_id}`,
                                    renderer: (row) => {
                                        return (
                                            <ButtonGroup>
                                                <Tooltip title={'Edit filter'}>
                                                    <IconButton
                                                        color={'warning'}
                                                        onClick={() => {
                                                            setEditFilter(row);
                                                        }}
                                                    >
                                                        <EditIcon />
                                                    </IconButton>
                                                </Tooltip>
                                                <Tooltip
                                                    title={'Delete filter'}
                                                >
                                                    <IconButton
                                                        color={'error'}
                                                        onClick={() => {
                                                            setDeleteTarget(
                                                                row
                                                            );
                                                            setDeleteModalOpen(
                                                                true
                                                            );
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
                            defaultSortColumn={'created_on'}
                            rowsPerPage={RowsPerPage.TwentyFive}
                            rows={filters}
                        />
                    </Paper>
                </Grid>
            </Grid>
        </>
    );
};
