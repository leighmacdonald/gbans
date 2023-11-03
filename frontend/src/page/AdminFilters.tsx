import React, { useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddBoxIcon from '@mui/icons-material/AddBox';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { noop } from 'lodash-es';
import { apiGetFilters, Filter } from '../api/filters';
import { DataTable, RowsPerPage } from '../component/DataTable';
import { Heading } from '../component/Heading';
import { ModalFilterDelete, ModalFilterEditor } from '../component/modal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminFilters = () => {
    const [filters, setFilters] = useState<Filter[]>([]);
    const { sendFlash } = useUserFlashCtx();

    const reset = async (abortController?: AbortController) => {
        try {
            setFilters((await apiGetFilters(abortController)) ?? []);
        } catch (e) {
            logErr(e);
        }
    };

    useEffect(() => {
        const abortController = new AbortController();

        reset(abortController).then(noop);

        return () => abortController.abort();
    }, []);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ButtonGroup
                    variant="contained"
                    aria-label="outlined primary button group"
                >
                    <Button
                        startIcon={<AddBoxIcon />}
                        color={'success'}
                        onClick={async () => {
                            try {
                                const resp = await NiceModal.show<Filter>(
                                    ModalFilterEditor,
                                    {
                                        defaultPattern: '',
                                        defaultIsRegex: false
                                    }
                                );
                                sendFlash(
                                    'success',
                                    `Filter created successfully: ${resp.filter_id}`
                                );
                            } catch (e) {
                                sendFlash('error', `${e}`);
                            }
                        }}
                    >
                        New
                    </Button>
                </ButtonGroup>
            </Grid>
            <Grid xs={12}>
                <Paper elevation={1}>
                    <Heading>Word Filters</Heading>
                    <DataTable
                        filterFn={(query, rows) => {
                            if (!query) {
                                return rows;
                            }
                            return rows.filter((f) => {
                                if (f.is_regex) {
                                    // TODO cache regex compilation
                                    const r = new RegExp(f.pattern);
                                    return r.test(query);
                                }
                                return (
                                    (f.pattern as string).toLowerCase() == query
                                );
                            });
                        }}
                        columns={[
                            {
                                label: 'Pattern',
                                tooltip: 'Pattern',
                                sortKey: 'pattern',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) => `${o.filter_id}`,
                                renderer: (row) => {
                                    return row.pattern as string;
                                }
                            },
                            {
                                label: 'Regex',
                                tooltip: 'Regular Expression',
                                sortKey: 'is_regex',
                                sortable: false,
                                align: 'right',
                                renderer: (row) => {
                                    return row.is_regex ? 'true' : 'false';
                                }
                            },
                            {
                                label: 'Enabled',
                                tooltip: 'Filter enabled',
                                sortKey: 'is_enabled',
                                sortable: false,
                                align: 'right',
                                renderer: (row) => {
                                    return row.is_enabled ? 'true' : 'false';
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
                                queryValue: (o) => `${o.filter_id}`,
                                renderer: (row) => {
                                    return (
                                        <ButtonGroup>
                                            <Tooltip title={'Edit filter'}>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () => {
                                                        await NiceModal.show(
                                                            ModalFilterEditor,
                                                            {
                                                                filter: row
                                                            }
                                                        );
                                                    }}
                                                >
                                                    <EditIcon />
                                                </IconButton>
                                            </Tooltip>
                                            <Tooltip title={'Delete filter'}>
                                                <IconButton
                                                    color={'error'}
                                                    onClick={async () => {
                                                        await NiceModal.show(
                                                            ModalFilterDelete,
                                                            {
                                                                record: row
                                                            }
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
    );
};
