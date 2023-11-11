import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import FlagIcon from '@mui/icons-material/Flag';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiGetDemos, DemoFile } from '../api';
import { logErr } from '../util/errors';
import { humanFileSize, renderDateTime } from '../util/text';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { VCenterBox } from './VCenterBox';
import { FilterButtons } from './formik/FilterButtons';
import { MapNameField, MapNameFieldValidator } from './formik/MapNameField';
import { ServerIDsField, serverIDsValidator } from './formik/ServerIDsField';
import { SourceIdField, sourceIdValidator } from './formik/SourceIdField';

interface STVFormValues {
    source_id: string;
    server_ids: number[];
    map_name: string;
}

const validationSchema = yup.object({
    source_id: sourceIdValidator,
    server_ids: serverIDsValidator,
    map_name: MapNameFieldValidator
});

export const STVTable = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof DemoFile>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [demos, setDemos] = useState<DemoFile[]>([]);
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();

    const onSubmit = useCallback(
        async (values: STVFormValues) => {
            const abortController = new AbortController();
            setLoading(true);
            try {
                const resp = await apiGetDemos(
                    {
                        limit: rowPerPageCount,
                        offset: page * rowPerPageCount,
                        order_by: sortColumn,
                        desc: sortOrder == 'desc',
                        steam_id: values.source_id,
                        map_name: values.map_name,
                        server_ids: values.server_ids
                    },
                    abortController
                );
                setDemos(resp.data);
                setTotalRows(resp.count);
            } catch (e) {
                logErr(e);
            } finally {
                setLoading(false);
            }
            return () => abortController.abort();
        },
        [page, rowPerPageCount, sortColumn, sortOrder]
    );
    const iv = {
        source_id: '',
        server_ids: [],
        map_name: ''
    };

    useEffect(() => {
        onSubmit(iv).catch(logErr);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
        <Formik
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
            initialValues={iv}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <Grid container spacing={2}>
                        <Grid xs>
                            <ServerIDsField />
                        </Grid>
                        <Grid xs>
                            <MapNameField />
                        </Grid>
                        <Grid xs>
                            <SourceIdField />
                        </Grid>
                        <Grid xs>
                            <VCenterBox>
                                <FilterButtons />
                            </VCenterBox>
                        </Grid>
                    </Grid>
                </Grid>
                <Grid xs={12}>
                    <LazyTable
                        loading={loading}
                        showPager={true}
                        count={totalRows}
                        rows={demos}
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
                            event: React.ChangeEvent<
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
                                tooltip: 'Server',
                                label: 'Server',
                                sortKey: 'server_name_short',
                                align: 'left',
                                width: '150px'
                            },
                            {
                                tooltip: 'Created On',
                                label: 'Created On',
                                sortKey: 'created_on',
                                align: 'left',
                                width: '150px',
                                renderer: (row) => {
                                    return renderDateTime(row.created_on);
                                }
                            },

                            {
                                tooltip: 'Map',
                                label: 'Map',
                                sortKey: 'map_name',
                                align: 'left',
                                renderer: (row) => {
                                    const re = /^workshop\/(.+?)\.ugc\d+$/;
                                    const match = row.map_name.match(re);
                                    if (!match) {
                                        return row.map_name;
                                    }
                                    return match[1];
                                }
                            },
                            {
                                tooltip: 'Size',
                                label: 'Size',
                                sortKey: 'size',
                                align: 'left',
                                width: '100px',
                                renderer: (obj) => {
                                    return humanFileSize(obj.size);
                                }
                            },
                            {
                                tooltip: 'Total Downloads',
                                label: '#',
                                align: 'left',
                                sortKey: 'downloads',
                                width: '50px',
                                renderer: (row) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {row.downloads}
                                        </Typography>
                                    );
                                }
                            },
                            {
                                tooltip: 'Create Report From Demo',
                                label: 'RP',
                                virtual: true,
                                align: 'center',
                                virtualKey: 'report',
                                width: '40px',
                                renderer: (row) => {
                                    return (
                                        <IconButton
                                            color={'error'}
                                            onClick={() => {
                                                sessionStorage.setItem(
                                                    'demoName',
                                                    row.title
                                                );
                                                navigate('/report');
                                            }}
                                        >
                                            <FlagIcon />
                                        </IconButton>
                                    );
                                }
                            },
                            {
                                tooltip: 'Download',
                                label: 'DL',
                                virtual: true,
                                align: 'center',
                                virtualKey: 'download',
                                width: '40px',
                                renderer: (row) => {
                                    return (
                                        <IconButton
                                            component={Link}
                                            href={`${window.gbans.asset_url}/${window.gbans.bucket_demo}/${row.title}`}
                                            color={'primary'}
                                        >
                                            <FileDownloadIcon />
                                        </IconButton>
                                    );
                                }
                            }
                        ]}
                    />
                </Grid>
            </Grid>
        </Formik>
    );
};
