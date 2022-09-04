import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Heading } from '../component/Heading';
import TextField from '@mui/material/TextField';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import { DataTable, RowsPerPage } from '../component/DataTable';
import { apiGetFilters, Filter } from '../api/filters';

export const AdminFilters = () => {
    const [newFilter, setNewFilter] = useState('');
    const [testFilter, setTestFilter] = useState('');
    const [filters, setFilters] = useState<Filter[]>([]);
    useEffect(() => {
        apiGetFilters().then((resp) => {
            if (!resp.status) {
                return;
            }
            setFilters(resp.result || []);
        });
    }, []);
    return (
        <>
            <Grid container spacing={2} paddingTop={3}>
                <Grid item xs={6}>
                    <Paper elevation={1}>
                        <Stack spacing={2}>
                            <Heading>Filter Tester</Heading>
                            <TextField
                                variant={'filled'}
                                fullWidth
                                label={'Test Filter'}
                                value={testFilter}
                                onChange={(evt) => {
                                    setTestFilter(evt.target.value);
                                }}
                            />
                        </Stack>
                    </Paper>
                </Grid>
                <Grid item xs={6}>
                    <Paper elevation={1}>
                        <Stack spacing={2}>
                            <Heading>Filter Creator</Heading>
                            <TextField
                                variant={'filled'}
                                fullWidth
                                label={'New Filter Regex'}
                                value={newFilter}
                                onChange={(evt) => {
                                    setNewFilter(evt.target.value);
                                }}
                            />
                            <ButtonGroup>
                                <Button color={'success'} variant={'contained'}>
                                    Create Rule
                                </Button>
                            </ButtonGroup>
                        </Stack>
                    </Paper>
                </Grid>
            </Grid>
            <DataTable
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
                        queryValue: (o) => `${o.word_id}`
                    }
                ]}
                defaultSortColumn={'created_on'}
                rowsPerPage={RowsPerPage.TwentyFive}
                rows={filters}
            />
        </>
    );
};
