import React, { useState } from 'react';
import Paper from '@mui/material/Paper';
import { ServerMap } from '../component/ServerMap';
import { apiGetServerStates, ServerState } from '../api';
import { LatLngLiteral } from 'leaflet';
import { MapStateCtx, useMapStateCtx } from '../contexts/MapStateCtx';
import Stack from '@mui/material/Stack';
import { ServerList } from '../component/ServerList';
import { useTimer } from 'react-timer-hook';
import { ServerFilters } from '../component/ServerFilters';
import LinearProgress, {
    LinearProgressProps
} from '@mui/material/LinearProgress';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { sum } from 'lodash-es';
import Grid from '@mui/material/Grid';
import Container from '@mui/material/Container';
import { getDistance } from '../util/gis';

function LinearProgressWithLabel(
    props: LinearProgressProps & { value: number }
) {
    return (
        <Box display="flex" alignItems="center">
            <Box width="100%" mr={1}>
                <LinearProgress variant="determinate" {...props} />
            </Box>
            <Box minWidth={35}>
                <Typography
                    variant="body2"
                    color="textSecondary"
                >{`${Math.round(props.value)}%`}</Typography>
            </Box>
        </Box>
    );
}

export const ServerStats = () => {
    const { servers } = useMapStateCtx();
    const cap = servers.length * 24;
    const use = sum(servers.map((value) => value?.players?.length || 0));
    const regions = servers.reduce((acc, cv) => {
        if (!Object.hasOwn(acc, cv.region)) {
            acc[cv.region] = [];
        }
        acc[cv.region].push(cv);
        return acc;
    }, {} as Record<string, ServerState[]>);
    const keys = Object.keys(regions);
    keys.sort();
    if (servers.length === 0) {
        return <></>;
    }
    return (
        <Container component={Paper}>
            <Grid
                container
                direction="row"
                justifyContent="space-evenly"
                alignItems="flex-start"
                justifyItems={'left'}
                spacing={3}
                padding={3}
            >
                <Grid item xs={3} xl={4}>
                    <Typography
                        style={{ display: 'inline' }}
                        variant={'subtitle1'}
                        align={'center'}
                    >
                        Global: {use} / {cap}
                    </Typography>
                    <LinearProgressWithLabel
                        value={Math.round((use / cap) * 100)}
                    />
                </Grid>

                {keys.map((v) => {
                    const pSum = sum(
                        ((Object.hasOwn(regions, v) && regions[v]) || []).map(
                            (value) => value?.players?.length || 0
                        )
                    );
                    const pMax = sum(
                        ((Object.hasOwn(regions, v) && regions[v]) || []).map(
                            (value) => value?.max_players || 24
                        )
                    );
                    return (
                        <Grid item xs={3} xl={4} key={`stat-${v}`}>
                            <Typography
                                style={{ display: 'inline' }}
                                variant={'subtitle1'}
                                align={'center'}
                            >
                                {v}: {pSum} / {pMax}
                            </Typography>
                            <LinearProgressWithLabel
                                value={Math.round((pSum / pMax) * 100)}
                            />
                        </Grid>
                    );
                })}
            </Grid>
        </Container>
    );
};

export const Servers = (): JSX.Element => {
    const [servers, setServers] = useState<ServerState[]>([]);
    const [pos, setPos] = useState<LatLngLiteral>({
        lat: 42.434719,
        lng: 42.434719
    });
    const [customRange, setCustomRange] = useState<number>(500);
    const [selectedServers, setSelectedServers] = useState<ServerState[]>([]);
    const [filterByRegion, setFilterByRegion] = useState<boolean>(false);
    const [showOpenOnly, setShowOpenOnly] = useState<boolean>(false);
    const [selectedRegion, setSelectedRegion] = useState<string>('any');

    const interval = 5;

    const nextExpiry = () => {
        const t0 = new Date();
        t0.setSeconds(t0.getSeconds() + interval);
        return t0;
    };

    const { restart } = useTimer({
        autoStart: true,
        expiryTimestamp: new Date(),
        onExpire: () => {
            apiGetServerStates()
                .then((response) => {
                    if (!response.status || !response.result) {
                        return;
                    }
                    setServers(
                        (response.result || [])
                            .filter((s) => s.enabled) // Shouldn't actually happen
                            .map((srv) => {
                                return {
                                    ...srv,
                                    distance: getDistance(pos, {
                                        lat: srv.latitude,
                                        lng: srv.longitude
                                    })
                                };
                            })
                    );
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
                customRange,
                setCustomRange,
                pos,
                setPos,
                selectedServers,
                setSelectedServers,
                filterByRegion,
                setFilterByRegion,
                showOpenOnly,
                setShowOpenOnly,
                selectedRegion,
                setSelectedRegion
            }}
        >
            <Stack spacing={3} paddingTop={3}>
                <Paper elevation={3}>
                    <ServerMap />
                </Paper>
                <ServerFilters />
                <Paper elevation={1}>
                    <ServerList />
                </Paper>
                <ServerStats />
            </Stack>
        </MapStateCtx.Provider>
    );
};
