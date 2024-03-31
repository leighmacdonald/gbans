import { ChangeEvent, useState } from 'react';
import SearchIcon from '@mui/icons-material/Search';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { PersonConnection } from '../api';
import { useConnections } from '../hooks/useConnections.ts';
import { Order, RowsPerPage } from '../util/table.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { TargetIDField, TargetIDInputValue } from './formik/TargetIdField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';
import { LazyTable } from './table/LazyTable.tsx';
import { connectionColumns } from './table/connectionColumns.tsx';

export const FindPlayerIPs = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof PersonConnection>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Ten
    );
    const [steamID, setSteamID] = useState('');
    const [page, setPage] = useState(0);

    const {
        data: rows,
        count,
        loading
    } = useConnections({
        limit: rowPerPageCount,
        offset: page * rowPerPageCount,
        desc: sortOrder == 'desc',
        order_by: sortColumn,
        source_id: steamID,
        asn: 0,
        cidr: ''
    });

    const onSubmit = (values: TargetIDInputValue) => {
        setSteamID(values.target_id);
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Formik onSubmit={onSubmit} initialValues={{ target_id: '' }}>
                    <Grid
                        container
                        direction="row"
                        alignItems="top"
                        justifyContent="center"
                        spacing={2}
                    >
                        <Grid xs>
                            <TargetIDField />
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
                        page={page}
                        rowsPerPage={rowPerPageCount}
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        onPageChange={(_, newPage: number) => {
                            setPage(newPage);
                        }}
                        onRowsPerPageChange={(
                            event: ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        columns={connectionColumns}
                    />
                )}
            </Grid>
        </Grid>
    );
};
