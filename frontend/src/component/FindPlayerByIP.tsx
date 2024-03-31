import { ChangeEvent, useState } from 'react';
import SearchIcon from '@mui/icons-material/Search';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { PersonConnection } from '../api';
import { useConnections } from '../hooks/useConnections.ts';
import { Order, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import {
    CIDRInputFieldProps,
    NetworkRangeField
} from './formik/NetworkRangeField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';
import { LazyTable } from './table/LazyTable.tsx';

export const FindPlayerByIP = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof PersonConnection>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Ten
    );
    const [cidr, setCIDR] = useState('');
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
        cidr: cidr,
        asn: 0,
        source_id: ''
    });

    const onSubmit = (values: CIDRInputFieldProps) => {
        setCIDR(values.cidr);
    };

    return (
        <Grid container>
            <Grid xs={12}>
                <Formik onSubmit={onSubmit} initialValues={{ cidr: '' }}>
                    <Grid
                        container
                        direction="row"
                        alignItems="top"
                        justifyContent="center"
                        spacing={2}
                    >
                        <Grid xs>
                            <NetworkRangeField />
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
                        columns={[
                            {
                                label: 'Created',
                                tooltip: 'Created On',
                                sortKey: 'created_on',
                                sortType: 'date',
                                align: 'left',
                                width: '150px',
                                sortable: true,
                                renderer: (obj: PersonConnection) => (
                                    <Typography variant={'body1'}>
                                        {renderDateTime(obj.created_on)}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Name',
                                tooltip: 'Name',
                                sortKey: 'persona_name',
                                sortType: 'string',
                                align: 'left',
                                width: '200px',
                                sortable: true
                            },
                            {
                                label: 'SteamID',
                                tooltip: 'Name',
                                sortKey: 'steam_id',
                                sortType: 'string',
                                align: 'left',
                                width: '200px',
                                sortable: true
                            },
                            {
                                label: 'IP Address',
                                tooltip: 'IP Address',
                                sortKey: 'ip_addr',
                                sortType: 'string',
                                align: 'left',
                                width: '150px',
                                sortable: true
                            },
                            {
                                label: 'Server',
                                tooltip: 'IP Address',
                                sortKey: 'ip_addr',
                                sortType: 'string',
                                align: 'left',
                                sortable: true,
                                renderer: (obj: PersonConnection) => {
                                    return (
                                        <Tooltip
                                            title={obj.server_name ?? 'Unknown'}
                                        >
                                            <Typography variant={'body1'}>
                                                {obj.server_name_short ??
                                                    'Unknown'}
                                            </Typography>
                                        </Tooltip>
                                    );
                                }
                            }
                        ]}
                    />
                )}
            </Grid>
        </Grid>
    );
};
