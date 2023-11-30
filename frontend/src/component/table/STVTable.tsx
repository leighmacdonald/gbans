import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import FlagIcon from '@mui/icons-material/Flag';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiGetDemos, DemoFile } from '../../api';
import { useCurrentUserCtx } from '../../contexts/CurrentUserCtx';
import { logErr } from '../../util/errors';
import { humanFileSize, renderDateTime } from '../../util/text';
import { emptyOrNullString } from '../../util/types';
import { FilterButtons } from '../formik/FilterButtons';
import { MapNameField, mapNameFieldValidator } from '../formik/MapNameField';
import { SelectOwnField, selectOwnValidator } from '../formik/SelectOwnField';
import { ServerIDsField, serverIDsValidator } from '../formik/ServerIDsField';
import { SourceIdField, sourceIdValidator } from '../formik/SourceIdField';
import { LazyTable, Order, RowsPerPage } from './LazyTable';

interface STVFormValues {
    source_id: string;
    server_ids: number[];
    map_name: string;
    select_own: boolean;
}

const validationSchema = yup.object({
    source_id: sourceIdValidator,
    server_ids: serverIDsValidator,
    map_name: mapNameFieldValidator,
    select_own: selectOwnValidator
});

export const STVTable = () => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof DemoFile>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [source, setSource] = useState('');
    const [mapName, setMapName] = useState('');
    const [serverIds, setServerIds] = useState<number[]>();
    const [selectOwn, setSelectOwn] = useState(false);
    const [demos, setDemos] = useState<DemoFile[]>([]);
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();
    const { currentUser } = useCurrentUserCtx();

    const selectOwnDisabled = useMemo(() => {
        return currentUser.steam_id == '';
    }, [currentUser.steam_id]);

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        let sourceID = '';
        if (selectOwn) {
            sourceID = currentUser.steam_id;
        } else if (!emptyOrNullString(source)) {
            sourceID = source;
        }
        apiGetDemos(
            {
                limit: rowPerPageCount,
                offset: page * rowPerPageCount,
                order_by: sortColumn,
                desc: sortOrder == 'desc',
                steam_id: sourceID,
                map_name: mapName,
                server_ids: serverIds ?? []
            },
            abortController
        )
            .then((resp) => {
                setDemos(resp.data);
                setTotalRows(resp.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [
        currentUser.steam_id,
        mapName,
        page,
        rowPerPageCount,
        selectOwn,
        serverIds,
        sortColumn,
        sortOrder,
        source
    ]);

    const iv: STVFormValues = {
        source_id: '',
        server_ids: [],
        map_name: '',
        select_own: false
    };

    const onReset = useCallback(() => {
        setSource(iv.source_id);
        setMapName(iv.map_name);
        setServerIds(iv.server_ids);
        setSelectOwn(iv.select_own);
    }, [iv.map_name, iv.select_own, iv.server_ids, iv.source_id]);

    const onSubmit = useCallback((values: STVFormValues) => {
        setSource(values.source_id);
        setMapName(values.map_name);
        setServerIds(values.server_ids);
        setSelectOwn(values.select_own);
    }, []);

    return (
        <Formik<STVFormValues>
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
            initialValues={iv}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <Grid container spacing={2}>
                        <Grid md>
                            <ServerIDsField />
                        </Grid>
                        <Grid md>
                            <MapNameField />
                        </Grid>
                        <Grid md>
                            <SourceIdField disabled={selectOwn} />
                        </Grid>
                        <Grid md>
                            <SelectOwnField disabled={selectOwnDisabled} />
                        </Grid>
                        <Grid md>
                            <FilterButtons />
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
