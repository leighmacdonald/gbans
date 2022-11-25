import React, { useMemo } from 'react';
import { apiGetDemos, DemoFile } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import { useCallback, useEffect, useState } from 'react';
import { DataTable, RowsPerPage } from './DataTable';
import { humanFileSize, renderDateTime } from '../util/text';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { ServerSelect } from './ServerSelect';
import Stack from '@mui/material/Stack';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import FormGroup from '@mui/material/FormGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import Checkbox from '@mui/material/Checkbox';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';

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

    const reload = useCallback(() => {
        setIsLoading(true);
        apiGetDemos({ steamId, mapName, serverIds })
            .then((response) => {
                setDemos(response.result ?? []);
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [mapName, serverIds, steamId]);

    useEffect(() => {
        reload();
    }, [reload]);

    const loggedIn = useMemo(() => {
        return currentUser.steam_id.isValidIndividual();
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
                                        setSteamId(
                                            currentUser.steam_id.getSteamID64()
                                        );
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
                            tooltip: 'Download',
                            label: 'DL',
                            virtual: true,
                            align: 'center',
                            virtualKey: 'Download',
                            width: '50px',
                            renderer: (row) => {
                                return (
                                    <IconButton
                                        component={Link}
                                        href={`/demos/${row.demo_id}`}
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
