import React, { useState } from 'react';
import { useTimer } from 'react-timer-hook';
import Box from '@mui/material/Box';
import Container from '@mui/material/Container';
import LinearProgress, {
    LinearProgressProps
} from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { LatLngLiteral } from 'leaflet';
import { sum } from 'lodash-es';
import { apiGetServerStates, BaseServer } from '../api';
import { ServerFilters } from '../component/ServerFilters';
import { ServerList } from '../component/ServerList';
import { ServerMap } from '../component/ServerMap';
import { MapStateCtx, useMapStateCtx } from '../contexts/MapStateCtx';

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
    const use = sum(servers.map((value) => value?.players || 0));
    const regions = servers.reduce(
        (acc, cv) => {
            if (!Object.prototype.hasOwnProperty.call(acc, cv.region)) {
                acc[cv.region] = [];
            }
            acc[cv.region].push(cv);
            return acc;
        },
        {} as Record<string, BaseServer[]>
    );
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
                <Grid xs={3} xl={4}>
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
                        (
                            (Object.prototype.hasOwnProperty.call(regions, v) &&
                                regions[v]) ||
                            []
                        ).map((value) => value?.players || 0)
                    );
                    const pMax = sum(
                        (
                            (Object.prototype.hasOwnProperty.call(regions, v) &&
                                regions[v]) ||
                            []
                        ).map((value) => value?.max_players || 24)
                    );
                    return (
                        <Grid xs={3} xl={4} key={`stat-${v}`}>
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

export const Servers = () => {
    const [servers, setServers] = useState<BaseServer[]>([]);
    const [pos, setPos] = useState<LatLngLiteral>({
        lat: 0.0,
        lng: 0.0
    });
    const [customRange, setCustomRange] = useState<number>(500);
    const [selectedServers, setSelectedServers] = useState<BaseServer[]>([]);
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
                    if (!response) {
                        restart(nextExpiry());
                        return;
                    }
                    setServers(response.servers || []);
                    if (pos.lat == 0) {
                        setPos({
                            lat: response.lat_long.latitude,
                            lng: response.lat_long.longitude
                        });
                    }

                    restart(nextExpiry());
                })
                .catch(() => {
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
            <Stack spacing={3}>
                <Paper elevation={3}>
                    <ServerMap />
                </Paper>
                <ServerFilters />
                <ServerList />
                <ServerStats />
            </Stack>
        </MapStateCtx.Provider>
    );
};
