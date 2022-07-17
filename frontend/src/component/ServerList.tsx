import Stack from '@mui/material/Stack';
import React from 'react';
import { ServerState } from '../api';
import Typography from '@mui/material/Typography';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import { Flag } from './Flag';
import LinearProgress from '@mui/material/LinearProgress';
import Box from '@mui/material/Box';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import { UserTable } from './UserTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { LinearProgressProps } from '@mui/material/LinearProgress';
import { LoadingSpinner } from './LoadingSpinner';

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

export interface ServerListProps {
    servers: ServerState[];
}

export interface ServerRowProps {
    server: ServerState;
}

export const ServerList = () => {
    const { sendFlash } = useUserFlashCtx();
    const { selectedServers } = useMapStateCtx();
    if (selectedServers.length === 0) {
        return (
            <Stack spacing={1}>
                <LoadingSpinner />
            </Stack>
        );
    }
    return (
        <UserTable
            rowsPerPage={100}
            columns={[
                {
                    label: 'CC',
                    tooltip: 'Country Code',
                    sortKey: 'cc',
                    sortType: 'string',
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
                    queryValue: (obj) => obj.map
                },
                {
                    label: 'Players',
                    tooltip: 'Current Players',
                    sortKey: 'player_count',
                    renderer: (obj, value) => {
                        return (
                            <Typography variant={'body2'}>
                                {`${value}/${obj.max_players}`}
                            </Typography>
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
                                aria-label={'Copy connect string to clipboard'}
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
            defaultSortColumn={'name'}
            rows={selectedServers}
        />
    );
};
