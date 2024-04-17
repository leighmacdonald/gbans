import { ChangeEvent } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import SearchIcon from '@mui/icons-material/Search';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { PersonConnection } from '../api';
import { useConnections } from '../hooks/useConnections.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import {
    CIDRInputFieldProps,
    NetworkRangeField
} from './formik/NetworkRangeField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';
import { LazyTable } from './table/LazyTable.tsx';

export const FindPlayesrByCIDR = () => {
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

    const onSubmit = (values: CIDRInputFieldProps) => {
        setState((prevState) => {
            return { ...prevState, cidr: values.cidr };
        });
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
