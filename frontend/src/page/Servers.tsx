import React, { useState } from 'react';
import Paper from '@mui/material/Paper';
import { ServerMap } from '../component/ServerMap';
import { apiGetServers, ServerState } from '../api';
import { LatLngLiteral } from 'leaflet';
import { MapStateCtx } from '../contexts/MapStateCtx';
import Stack from '@mui/material/Stack';
import { ServerList } from '../component/ServerList';
import { useTimer } from 'react-timer-hook';

export const Servers = (): JSX.Element => {
    const interval = 5;
    const [servers, setServers] = useState<ServerState[]>([]);
    const [pos, setPos] = useState<LatLngLiteral>({
        lat: 42.434719,
        lng: 42.434719
    });
    const nextExpiry = () => {
        const t0 = new Date();
        t0.setSeconds(t0.getSeconds() + interval);
        return t0;
    };
    const { restart } = useTimer({
        autoStart: true,
        expiryTimestamp: new Date(),
        onExpire: () => {
            apiGetServers()
                .then((servers) => {
                    setServers(servers);
                    restart(nextExpiry());
                })
                .catch((e) => {
                    alert(`Failed to load server: ${e}`);
                    restart(nextExpiry());
                });
        }
    });
    return (
        <MapStateCtx.Provider
            value={{
                servers,
                setServers,
                pos,
                setPos
            }}
        >
            <Stack spacing={3} paddingTop={3}>
                <Paper elevation={3}>
                    <ServerMap />
                </Paper>
                <Paper elevation={1}>
                    <ServerList servers={servers} />
                </Paper>
            </Stack>
        </MapStateCtx.Provider>
    );
};
