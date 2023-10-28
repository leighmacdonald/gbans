import React, { useMemo } from 'react';
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import FlagIcon from '@mui/icons-material/Flag';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { apiGetDemos, DemoFile } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { humanFileSize, renderDateTime } from '../util/text';
import { ContainerWithHeader } from './ContainerWithHeader';
import { DataTable, RowsPerPage } from './DataTable';
import { ServerSelect } from './ServerSelect';

export interface STVListVIewProps {
    demos: DemoFile[];
}

export const STVListVIew = () => {
    const [demos, setDemos] = useState<DemoFile[]>([]);
    const [steamId, setSteamId] = useState('');
    const [mapName] = useState('');
    const [serverIds, setServerIds] = useState<number[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const { currentUser } = useCurrentUserCtx();
    const navigate = useNavigate();

    const reload = useCallback(
        (abortController: AbortController) => {
            setIsLoading(true);
            apiGetDemos(
                {
                    steam_id: steamId,
                    map_name: mapName,
                    server_ids: serverIds
                },
                abortController
            )
                .then((response) => {
                    setDemos(response);
                })
                .catch((e) => {
                    logErr(e);
                })
                .finally(() => {
                    setIsLoading(false);
                });
        },
        [mapName, serverIds, steamId]
    );

    useEffect(() => {
        const abortController = new AbortController();
        reload(abortController);
        return abortController.abort();
    }, [reload]);

    const loggedIn = useMemo(() => {
        return currentUser.steam_id != '';
    }, [currentUser]);

    return (
        <Stack spacing={4}>
            <ContainerWithHeader title={'SourceTV Recordings'}>
                <Box paddingLeft={2} paddingRight={2}>
                    <Stack direction={'row'} spacing={2}>
                        <ServerSelect setServerIDs={setServerIds} />
                        <Tooltip
                            title={
                                loggedIn
                                    ? 'Filter to demos that you have participated in'
                                    : 'Please login to filter to your own demos'
                            }
                        >
                            <FormGroup>
                                <FormControlLabel
                                    control={<Checkbox />}
                                    label="Only&nbsp;Mine"
                                    disabled={!loggedIn}
                                    onChange={(_, checked) => {
                                        if (!checked || !loggedIn) {
                                            setSteamId('');
                                            return;
                                        }
                                        setSteamId(currentUser.steam_id);
                                    }}
                                />
                            </FormGroup>
                        </Tooltip>
                    </Stack>
                </Box>
                <DataTable
                    isLoading={isLoading}
                    columns={[
                        {
                            tooltip: 'Server',
                            label: 'Server',
                            sortKey: 'server_name_short',
                            align: 'left',
                            width: '150px',
                            queryValue: (v) => {
                                return v.server_name_long + v.server_name_short;
                            }
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
                            queryValue: (v) => {
                                return v.map_name;
                            },
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
                    defaultSortColumn={'created_on'}
                    rowsPerPage={RowsPerPage.Fifty}
                    rows={demos}
                />
            </ContainerWithHeader>
        </Stack>
    );
};
