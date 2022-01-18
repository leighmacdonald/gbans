import React, { useState } from 'react';
import { ServerList } from '../component/ServerList';
import { Grid, Paper } from '@mui/material';
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
                    <ServerMap />
                </Grid>
                <Grid item xs={12}>
                    <Paper>
                        <ServerList />
                    </Paper>
                </Grid>
            </Grid>
        </MapStateCtx.Provider>
    );
};
