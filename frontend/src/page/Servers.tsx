import React, { useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { ServerList } from '../component/ServerList';
import { ServerMap } from '../component/ServerMap';
import { Server } from '../util/api';
import { LatLngLiteral } from 'leaflet';
import { MapStateCtx } from '../contexts/MapStateCtx';

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
            <Grid container>
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <ServerMap />
                    </Paper>
                </Grid>
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <ServerList />
                    </Paper>
                </Grid>
            </Grid>
        </MapStateCtx.Provider>
    );
};
