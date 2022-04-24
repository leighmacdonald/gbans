import Stack from '@mui/material/Stack';
import React, { useEffect } from 'react';
import { apiGetServers, Server } from '../api';
import Typography from '@mui/material/Typography';
import { useMapStateCtx } from '../contexts/MapStateCtx';

export interface ServerListProps {
    servers: Server[];
}
export const ServerList = ({ servers }: ServerListProps) => {
    const { setServers } = useMapStateCtx();
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
        <Stack>
            {servers.map((server) => {
                return (
                    <Stack
                        direction={'column'}
                        key={`server-${server.server_id}`}
                    >
                        <Typography variant={'h3'}>
                            {server.server_name_long}
                        </Typography>
                    </Stack>
                );
            })}
        </Stack>
    );
};
