import React, { ChangeEvent, useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import { Heading } from '../component/Heading';
import ButtonGroup from '@mui/material/ButtonGroup';
import EditIcon from '@mui/icons-material/Edit';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import { DataTable, RowsPerPage } from '../component/DataTable';
import { apiGetFilters, apiSaveFilter, Filter } from '../api/filters';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { ConfirmDeleteFilterModal } from '../component/ConfirmDeleteFilterModal';
import { Nullable } from '../util/types';
import Button from '@mui/material/Button';
import AddBoxIcon from '@mui/icons-material/AddBox';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Switch from '@mui/material/Switch';
import { logErr } from '../util/errors';
import FormControlLabel from '@mui/material/FormControlLabel';
import Typography from '@mui/material/Typography';

interface FilterEditModalProps {
    open: boolean;
    setOpen: (openState: boolean) => void;
    filterId?: number;
    defaultPattern?: string;
    defaultIsRegex?: boolean;
    onSuccess: (filter: Filter) => void;
}

interface FilterTestFieldProps {
    pattern: string;
    isRegex: boolean;
}

const FilterTestField = ({ pattern, isRegex }: FilterTestFieldProps) => {
    const [testString, setTestString] = useState<string>('');
    const [matched, setMatched] = useState(false);
    const [validPattern, setValidPattern] = useState(false);

    useEffect(() => {
        if (!pattern) {
            setValidPattern(false);
            setMatched(false);
            return;
        }
        if (isRegex) {
            try {
                const p = new RegExp(pattern, 'g');
                setMatched(p.test(testString.toLowerCase()));
                setValidPattern(true);
            } catch (e) {
                setValidPattern(false);
                logErr(e);
            }
        } else {
            setMatched(pattern.toLowerCase() == testString.toLowerCase());
            setValidPattern(true);
        }
    }, [isRegex, pattern, testString]);

    return (
        <Stack>
            <TextField
                id="test-string"
                label="Test String"
                value={testString}
                onChange={(event) => {
                    setTestString(event.target.value);
                }}
            />
            {pattern && (
                <Typography
                    variant={'caption'}
                    color={validPattern && matched ? 'success' : 'error'}
                >
                    {validPattern
                        ? matched
                            ? 'Matched'
                            : 'No Match'
                        : 'Invalid Pattern'}
                </Typography>
            )}
        </Stack>
    );
};

const FilterEditModal = ({
    open,
    onSuccess,
    setOpen,
    filterId,
    defaultPattern = '',
    defaultIsRegex = false
}: FilterEditModalProps) => {
    const [isRegex, setIsRegex] = useState<boolean>(defaultIsRegex);
    const [pattern, setPattern] = useState<string>(defaultPattern);
    const handleClose = () => setOpen(false);

    const onSave = useCallback(async () => {
        const f: Filter = {
            is_enabled: true,
            filter_id: filterId,
            is_regex: isRegex,
            pattern: pattern
        };
        try {
            const resp = await apiSaveFilter(f);
            if (resp.result) {
                onSuccess(resp.result);
            }
        } catch (e) {
            logErr(e);
        }
    }, [filterId, isRegex, onSuccess, pattern]);
    return (
        <Dialog open={open} onClose={handleClose} fullWidth maxWidth={'sm'}>
            <DialogTitle component={Heading}>Filter Editor</DialogTitle>
            <DialogContent>
                <Stack spacing={2}>
                    <TextField
                        required
                        id="outlined-pattern"
                        label="Pattern"
                        value={pattern}
                        onChange={(event) => {
                            setPattern(event.target.value);
                        }}
                    />
                    <FormControlLabel
                        control={
                            <Switch
                                checked={isRegex}
                                onChange={(
                                    _: ChangeEvent<HTMLInputElement>,
                                    checked: boolean
                                ) => {
                                    setIsRegex(checked);
                                }}
                            />
                        }
                        label="Regular Expression"
                    />
                    <FilterTestField pattern={pattern} isRegex={isRegex} />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button
                    variant={'contained'}
                    color={'error'}
                    onClick={() => {
                        setOpen(false);
                    }}
                >
                    Cancel
                </Button>
                <Button
                    variant={'contained'}
                    color={'success'}
                    onClick={onSave}
                    disabled={!pattern}
                >
                    Save Filter
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export const AdminFilters = () => {
    const [filters, setFilters] = useState<Filter[]>([]);
    const [deleteTarget, setDeleteTarget] = useState<Nullable<Filter>>();
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);
    const [editorOpen, setEditorOpen] = useState<boolean>(false);

    const [selected, setSelected] = useState<Nullable<Filter>>();

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

    return (
        <>
            {editorOpen && (
                <FilterEditModal
                    open={editorOpen}
                    setOpen={setEditorOpen}
                    filterId={selected?.filter_id}
                    defaultIsRegex={selected?.is_regex}
                    defaultPattern={selected?.pattern as string}
                    onSuccess={() => {
                        reset();
                        setEditorOpen(false);
                    }}
                />
            )}
            {deleteTarget && (
                <ConfirmDeleteFilterModal
                    record={deleteTarget}
                    open={deleteModalOpen}
                    setOpen={setDeleteModalOpen}
                    onSuccess={() => {
                        setFilters(
                            filters.filter(
                                (f) => f.filter_id != deleteTarget?.filter_id
                            )
                        );
                        setDeleteModalOpen(false);
                    }}
                />
            )}
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ButtonGroup
                        variant="contained"
                        aria-label="outlined primary button group"
                    >
                        <Button
                            startIcon={<AddBoxIcon />}
                            color={'success'}
                            onClick={() => {
                                setSelected(null);
                                setEditorOpen(true);
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
                                        (f.pattern as string).toLowerCase() ==
                                        query
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
                                        return row.is_enabled
                                            ? 'true'
                                            : 'false';
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
                                                        onClick={() => {
                                                            setSelected(row);
                                                            setEditorOpen(true);
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
