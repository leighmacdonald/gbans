import Stack from '@mui/material/Stack';
import React, { useEffect } from 'react';
import { apiGetServers, Server } from '../api';
import Typography from '@mui/material/Typography';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import { Flag } from './Flag';
import { LinearProgress, LinearProgressProps, useTheme } from '@mui/material';
import Box from '@mui/material/Box';

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
    servers: Server[];
}

export const ServerList = ({ servers }: ServerListProps) => {
    const { setServers } = useMapStateCtx();
    const theme = useTheme();
    useEffect(() => {
        const fn = async () => {
            try {
                setServers(await apiGetServers());
            } catch (e) {
                alert('Failed to load server');
            }
        };
        fn();
    }, []);
    return (
        <Stack spacing={1}>
            {servers.map((server) => {
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
                                    backgroundColor:
                                        theme.palette.background.default
                                }
                            }
                        ]}
                    >
                        <Flag countryCode={server.cc} />
                        <Typography variant={'h4'} sx={{ width: '100%' }}>
                            {server.server_name_long}
                        </Typography>
                        <Typography
                            variant={'h5'}
                            sx={{ minWidth: 200 }}
                            align={'center'}
                        >
                            {server.current_map}
                        </Typography>
                        <Typography variant={'h5'} sx={{ minWidth: 100 }}>
                            {server.players?.length || 0} /{' '}
                            {Math.max(
                                24,
                                server.players_max - server.reserved_slots
                            )}
                        </Typography>
                        <IconButton
                            color={'success'}
                            component="span"
                            sx={{ width: 10 }}
                        >
                            <ContentCopyIcon />
                        </IconButton>
                        <Button variant={'contained'} sx={{ minWidth: 100 }}>
                            Connect
                        </Button>
                    </Stack>
                );
            })}
        </Stack>
    );
};
