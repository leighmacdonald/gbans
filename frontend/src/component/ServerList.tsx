import React from 'react';
import Typography from '@mui/material/Typography';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Tooltip from '@mui/material/Tooltip';
import { Flag } from './Flag';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { LoadingSpinner } from './LoadingSpinner';
import { DataTable, RowsPerPage } from './DataTable';
import { cleanMapName } from '../api';
import { tf2Fonts } from '../theme';
import { ContainerWithHeader } from './ContainerWithHeader';
import StorageIcon from '@mui/icons-material/Storage';
import { logErr } from '../util/errors';

export const ServerList = () => {
    const { sendFlash } = useUserFlashCtx();
    const { selectedServers } = useMapStateCtx();
    if (selectedServers.length === 0) {
        return (
            <ContainerWithHeader
                title={'Servers Loading...'}
                iconLeft={<StorageIcon />}
            >
                <LoadingSpinner />
            </ContainerWithHeader>
        );
    }
    return (
        <ContainerWithHeader title={'Servers'} iconLeft={<StorageIcon />}>
            <DataTable
                defaultSortOrder={'asc'}
                rowsPerPage={RowsPerPage.Hundred}
                columns={[
                    {
                        label: 'CC',
                        tooltip: 'Country Code',
                        sortKey: 'cc',
                        sortType: 'string',
                        sortable: true,
                        queryValue: (obj) => obj.cc,
                        renderer: (_, value) => (
                            <Flag countryCode={value as string} />
                        )
                    },
                    {
                        label: 'Server',
                        tooltip: 'Server Name',
                        sortKey: 'name',
                        sortType: 'string',
                        sortable: true,
                        align: 'left',
                        width: '100%',
                        queryValue: (obj) => obj.name + obj.name_short,
                        renderer: (_, value) => (
                            <Typography
                                variant={'button'}
                                fontFamily={tf2Fonts}
                            >
                                {value as string}
                            </Typography>
                        )
                    },
                    {
                        label: 'Map',
                        tooltip: 'Map Name',
                        sortKey: 'map',
                        sortType: 'string',
                        sortable: true,
                        queryValue: (obj) => obj.map,
                        renderer: (obj) => {
                            return (
                                <Typography variant={'body2'}>
                                    {cleanMapName(obj.map)}
                                </Typography>
                            );
                        }
                    },
                    {
                        label: 'Players',
                        tooltip: 'Current Players',
                        sortKey: 'players',
                        sortable: true,
                        renderer: (obj, value) => {
                            return (
                                <Typography variant={'body2'}>
                                    {`${value}/${obj.max_players}`}
                                </Typography>
                            );
                        }
                    },
                    {
                        label: 'Dist',
                        tooltip: `Distance to the server`,
                        sortKey: 'distance',
                        sortable: true,
                        renderer: (obj) => {
                            return (
                                <Tooltip
                                    title={`Distance in hammer units: ${Math.round(
                                        (obj.distance ?? 1) * 52.49
                                    )} khu`}
                                >
                                    <Typography variant={'caption'}>
                                        {`${obj.distance.toFixed(0)}km`}
                                    </Typography>
                                </Tooltip>
                            );
                        }
                    },
                    {
                        label: 'Cp',
                        virtual: true,
                        virtualKey: 'copy',
                        tooltip: 'Copy server address to clipboard',
                        renderer: (obj) => {
                            return (
                                <IconButton
                                    color={'primary'}
                                    aria-label={
                                        'Copy connect string to clipboard'
                                    }
                                    onClick={() => {
                                        navigator.clipboard
                                            .writeText(
                                                `connect ${obj.host}:${obj.port}`
                                            )
                                            .then(() => {
                                                sendFlash(
                                                    'success',
                                                    'Copied address to clipboard'
                                                );
                                            })
                                            .catch((e) => {
                                                sendFlash(
                                                    'error',
                                                    'Failed to copy address'
                                                );
                                                logErr(e);
                                            });
                                    }}
                                >
                                    <ContentCopyIcon />
                                </IconButton>
                            );
                        }
                    },
                    {
                        label: 'Connect',
                        virtual: true,
                        virtualKey: 'connect',
                        tooltip: 'Connect to a server',
                        renderer: (serverState) => {
                            return (
                                <Button
                                    component={Link}
                                    href={`steam://connect/${serverState.ip}:${serverState.port}`}
                                    variant={'contained'}
                                    sx={{ minWidth: 100 }}
                                >
                                    Join
                                </Button>
                            );
                        }
                    }
                ]}
                defaultSortColumn={'distance'}
                rows={selectedServers}
            />
        </ContainerWithHeader>
    );
};
