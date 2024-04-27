import { ChangeEvent, useCallback, useMemo } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import FlagIcon from '@mui/icons-material/Flag';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useNavigate, useRouteContext } from '@tanstack/react-router';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import * as yup from 'yup';
import { useDemos } from '../../hooks/useDemos';
import { RowsPerPage } from '../../util/table.ts';
import { humanFileSize, renderDateTime } from '../../util/text.tsx';
import { emptyOrNullString } from '../../util/types';
import { mapNameFieldValidator, selectOwnValidator, serverIDsValidator, sourceIdValidator } from '../../util/validators.ts';
import { FilterButtons } from '../formik/FilterButtons';
import { MapNameField } from '../formik/MapNameField.tsx';
import { SelectOwnField } from '../formik/SelectOwnField.tsx';
import { ServerIDsField } from '../formik/ServerIDsField.tsx';
import { SourceIDField } from '../formik/SourceIDField.tsx';
import { LazyTable } from './LazyTable';

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
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        own: undefined,
        mapName: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined,
        serverIds: undefined
    });
    const navigate = useNavigate();
    const { userSteamID } = useRouteContext({ from: '/_authoptional/stv' });

    const selectOwnDisabled = useMemo(() => {
        return userSteamID == '';
    }, [userSteamID]);

    const { data, count } = useDemos({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'created_on',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        steam_id: state.own ? userSteamID : !emptyOrNullString(state.source) ? state.source : '',
        map_name: state.mapName ?? '',
        server_ids: state.serverIds ?? undefined
    });

    const onReset = useCallback(
        async (_: STVFormValues, formikHelpers: FormikHelpers<STVFormValues>) => {
            setState({
                own: undefined,
                source: undefined,
                target: undefined,
                mapName: undefined,
                serverIds: []
            });

            await formikHelpers.setFieldValue('map_name', undefined);
            await formikHelpers.setFieldValue('source_id', undefined);
            await formikHelpers.setFieldValue('server_ids', []);
            await formikHelpers.setFieldValue('select_own', false);
            await formikHelpers.setTouched({ select_own: true }, false);
        },
        [setState]
    );

    const onSubmit = useCallback(
        (values: STVFormValues) => {
            setState({
                own: values.select_own ? values.select_own : undefined,
                source: values.source_id != '' ? values.source_id : undefined,
                mapName: values.map_name != '' ? values.map_name : undefined,
                serverIds: (values.server_ids ?? []).length > 0 ? values.server_ids.map((i) => Number(i)) : []
            });
        },
        [setState]
    );

    return (
        <Formik<STVFormValues>
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
            initialValues={{
                source_id: state.source,
                server_ids: state.serverIds ? state.serverIds : [],
                map_name: state.mapName,
                select_own: Boolean(state.own)
            }}
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
                            <SourceIDField disabled={Boolean(state.own)} />
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
                        showPager={true}
                        count={count}
                        rows={data}
                        page={Number(state.page ?? 0)}
                        rowsPerPage={Number(state.rows ?? RowsPerPage.TwentyFive)}
                        sortOrder={state.sortOrder}
                        sortColumn={state.sortColumn}
                        onSortColumnChanged={async (column) => {
                            setState({ sortColumn: column });
                        }}
                        onSortOrderChanged={async (direction) => {
                            setState({ sortOrder: direction });
                        }}
                        onPageChange={(_, newPage: number) => {
                            setState({ page: newPage });
                        }}
                        onRowsPerPageChange={(event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                            setState({
                                rows: Number(event.target.value),
                                page: 0
                            });
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
                                hideSm: true,
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
                                    return <Typography variant={'body1'}>{row.downloads}</Typography>;
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
                                                sessionStorage.setItem('demoName', row.title);
                                                navigate({ to: '/report' });
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
