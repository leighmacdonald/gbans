import Stack from '@mui/material/Stack';
import React from 'react';
import Typography from '@mui/material/Typography';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import { Flag } from './Flag';
import LinearProgress from '@mui/material/LinearProgress';
import Box from '@mui/material/Box';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { LinearProgressProps } from '@mui/material/LinearProgress';
import { LoadingSpinner } from './LoadingSpinner';
import { DataTable } from './DataTable';
import { Heading } from './Heading';
import { cleanMapName } from '../api';
import Tooltip from '@mui/material/Tooltip';

export const LinearProgressWithLabel = (
    props: LinearProgressProps & { value: number }
) => (
    <Box display="flex" alignItems="center">
        <Box width="100%" mr={1}>
            <LinearProgress variant="determinate" {...props} />
        </Box>
        <Box minWidth={35}>
            <Typography variant="body2" color="textSecondary">{`${Math.round(
                props.value
            )}%`}</Typography>
        </Box>
    </Box>
);

export const ServerList = () => {
    const { sendFlash } = useUserFlashCtx();
    const { selectedServers } = useMapStateCtx();
    if (selectedServers.length === 0) {
        return (
            <Stack spacing={1}>
                <Heading>Servers</Heading>
                <LoadingSpinner />
            </Stack>
        );
    }
    return (
        <Stack spacing={1}>
            <Heading>Servers</Heading>
            <DataTable
                defaultSortOrder={'asc'}
                rowsPerPage={100}
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
                            <Typography variant={'button'}>
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
                        sortKey: 'player_count',
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
                        tooltip: () => `Distance to the server`,
                        sortKey: 'distance',
                        sortable: true,
                        renderer: (obj) => {
                            return (
                                <Tooltip
                                    title={`Distance in hammer units: ${Math.round(
                                        (obj.distance ?? 1) * 52.49
                                    )} khu`}
                                >
                                    <Typography variant={'body2'}>
                                        {`${obj.distance}km`}
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
                                            .catch(() => {
                                                sendFlash(
                                                    'error',
                                                    'Failed to copy address'
                                                );
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
                        renderer: (obj) => {
                            return (
                                <Button
                                    onClick={() => {
                                        window.open(
                                            `steam://connect/${obj.host}:${obj.port}`
                                        );
                                    }}
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
        </Stack>
    );
};
