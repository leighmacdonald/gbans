import { ChangeEvent } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import SearchIcon from '@mui/icons-material/Search';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { PersonConnection } from '../api';
import { useConnections } from '../hooks/useConnections.ts';
import { RowsPerPage } from '../util/table.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { SourceIDField, SourceIDFieldValue } from './formik/SourceIDField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';
import { LazyTable } from './table/LazyTable.tsx';
import { connectionColumns } from './table/connectionColumns.tsx';

export const FindPlayerIPs = () => {
    const [state, setState] = useUrlState({
        page: undefined,
        source_id: undefined,
        asn: undefined,
        cidr: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const {
        data: rows,
        count,
        loading
    } = useConnections({
        limit: state.rows ?? RowsPerPage.TwentyFive,
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'created_on',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source_id ?? '',
        asn: 0,
        cidr: state.cidr ?? ''
    });

    const onSubmit = (values: SourceIDFieldValue) => {
        setState((prevState) => {
            return { ...prevState, source_id: values.source_id };
        });
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Formik onSubmit={onSubmit} initialValues={{ source_id: '' }}>
                    <Grid
                        container
                        direction="row"
                        alignItems="top"
                        justifyContent="center"
                        spacing={2}
                    >
                        <Grid xs>
                            <SourceIDField />
                        </Grid>
                        <Grid xs={2}>
                            <SubmitButton
                                label={'Submit'}
                                fullWidth
                                disabled={loading}
                                startIcon={<SearchIcon />}
                            />
                        </Grid>
                    </Grid>
                </Formik>
            </Grid>
            <Grid xs={12}>
                {loading ? (
                    <LoadingPlaceholder />
                ) : (
                    <LazyTable<PersonConnection>
                        showPager={true}
                        count={count}
                        rows={rows}
                        page={state.page}
                        rowsPerPage={state.rows}
                        sortOrder={state.sortOrder}
                        sortColumn={state.sortColumn}
                        onSortColumnChanged={async (column) => {
                            setState((prevState) => {
                                return { ...prevState, sortColumn: column };
                            });
                        }}
                        onSortOrderChanged={async (direction) => {
                            setState((prevState) => {
                                return { ...prevState, sortOrder: direction };
                            });
                        }}
                        onPageChange={(_, newPage: number) => {
                            setState((prevState) => {
                                return { ...prevState, page: newPage };
                            });
                        }}
                        onRowsPerPageChange={(
                            event: ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setState((prevState) => {
                                return {
                                    ...prevState,
                                    rows: parseInt(event.target.value, 10),
                                    page: 0
                                };
                            });
                        }}
                        columns={connectionColumns}
                    />
                )}
            </Grid>
        </Grid>
    );
};
