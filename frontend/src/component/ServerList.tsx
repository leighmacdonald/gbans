import Stack from '@mui/material/Stack';
import React, { useState, MouseEvent } from 'react';
import { ServerState } from '../api';
import Typography from '@mui/material/Typography';
import CheckIcon from '@mui/icons-material//Check';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import { Flag } from './Flag';
import {
    LinearProgress,
    LinearProgressProps,
    Popover,
    Tooltip,
    useTheme
} from '@mui/material';
import Box from '@mui/material/Box';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import { getDistance } from '../util/gis';
import Link from '@mui/material/Link';

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

export const ServerRow = ({ server }: ServerRowProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const [copied, setCopied] = useState<boolean>(false);

    const handlePopoverOpen = (event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handlePopoverClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    return (
        <Stack
            alignItems={'center'}
            spacing={1}
            direction={'row'}
            padding={1}
            key={`server-${server.server_id}`}
            sx={[
                {
                    '&:hover': {
                        backgroundColor: theme.palette.background.default
                    }
                }
            ]}
        >
            <Tooltip title={server.cc}>
                <Box>
                    <Flag countryCode={server.cc} />
                    <Typography variant={'h5'} sx={{ width: '100%' }}>
                        {server.name}
                    </Typography>
                </Box>
            </Tooltip>

            <div>
                <Typography
                    aria-owns={open ? 'mouse-over-popover' : undefined}
                    aria-haspopup="true"
                    onMouseEnter={handlePopoverOpen}
                    onMouseLeave={handlePopoverClose}
                    variant={'h6'}
                    sx={{ minWidth: 200 }}
                    align={'center'}
                >
                    {server.map}
                </Typography>
                <Popover
                    id="mouse-over-popover"
                    sx={{
                        pointerEvents: 'none'
                    }}
                    open={open}
                    anchorEl={anchorEl}
                    anchorOrigin={{
                        vertical: 'top',
                        horizontal: 'left'
                    }}
                    transformOrigin={{
                        vertical: 'top',
                        horizontal: 'left'
                    }}
                    onClose={handlePopoverClose}
                    disableRestoreFocus
                >
                    <Link
                        target={
                            'https://wiki.teamfortress.com/wiki/' + server.map
                        }
                    >
                        <Typography sx={{ p: 1 }}>{server.map}</Typography>
                    </Link>
                </Popover>
            </div>
            <Typography variant={'h6'} sx={{ minWidth: 100 }}>
                {server.players?.length || 0} /{' '}
                {Math.max(24, server.max_players - server.reserved)}
            </Typography>
            <Tooltip title={'Copy connect info to clipboard'}>
                <IconButton
                    color={'primary'}
                    aria-label={'Copy connect string to clipboard'}
                    onClick={() => {
                        navigator.clipboard
                            .writeText(`connect ${server.host}:${server.port}`)
                            .then(() => {
                                setCopied(true);
                            })
                            .catch(() => {
                                setCopied(false);
                            });
                    }}
                >
                    {copied ? <CheckIcon /> : <ContentCopyIcon />}
                </IconButton>
            </Tooltip>
            <Tooltip title={'Connect to server'}>
                <Button variant={'contained'} sx={{ minWidth: 100 }}>
                    Join
                </Button>
            </Tooltip>
        </Stack>
    );
};

export const ServerList = () => {
    const { selectedServers, pos } = useMapStateCtx();
    if (selectedServers.length === 0) {
        return (
            <Stack spacing={1}>
                <Typography variant={'h1'}>No servers :&apos;(</Typography>
            </Stack>
        );
    }
    return (
        <Stack spacing={1}>
            {(selectedServers || [])
                .map((srv) => {
                    return { ...srv, distance: getDistance(pos, srv.location) };
                })
                .sort((a, b) => {
                    // Sort by position if we have a non-default position.
                    // otherwise, sort by server name
                    if (pos.lat !== 42.434719) {
                        if (a.distance > b.distance) {
                            return 1;
                        }
                        if (a.distance < b.distance) {
                            return -1;
                        }
                        return 0;
                    }
                    return ('' + a.name_short).localeCompare(b.name_short);
                })
                .map((server) => (
                    <ServerRow server={server} key={server.server_id} />
                ))}
        </Stack>
    );
};
