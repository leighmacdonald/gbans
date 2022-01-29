import React, { useState } from 'react';
import Paper from '@mui/material/Paper';
import { ServerList } from '../component/ServerList';
import { ServerMap } from '../component/ServerMap';
import { Server } from '../api';
import { LatLngLiteral } from 'leaflet';
import { MapStateCtx } from '../contexts/MapStateCtx';
import Stack from '@mui/material/Stack';

export const Servers = (): JSX.Element => {
    const [servers, setServers] = useState<Server[]>([]);
    const [pos, setPos] = useState<LatLngLiteral>({
        lat: 42.434719,
        lng: 42.434719
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
                <Paper elevation={1}>
                    <ServerMap />
                </Paper>
                <Paper elevation={1}>
                    <ServerList />
                </Paper>
            </Stack>
        </MapStateCtx.Provider>
    );
};
