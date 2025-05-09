import { useMemo, useState } from 'react';
import { useTimer } from 'react-timer-hook';
import HelpIcon from '@mui/icons-material/Help';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import StorageIcon from '@mui/icons-material/Storage';
import Box from '@mui/material/Box';
import Container from '@mui/material/Container';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import LinearProgress, { LinearProgressProps } from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { createFileRoute } from '@tanstack/react-router';
import { LatLngLiteral } from 'leaflet';
import { apiGetServerStates, BaseServer, PermissionLevel } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { ServerFilters } from '../component/ServerFilters.tsx';
import { ServerList } from '../component/ServerList.tsx';
import { ServerMap } from '../component/ServerMap.tsx';
import { Title } from '../component/Title.tsx';
import { QueueHelp } from '../component/queue/QueueHelp.tsx';
import { MapStateCtx } from '../contexts/MapStateCtx.tsx';
import { useAuth } from '../hooks/useAuth.ts';
import { useMapStateCtx } from '../hooks/useMapStateCtx.ts';
import { ensureFeatureEnabled } from '../util/features.ts';
import { sum } from '../util/lists.ts';

export const Route = createFileRoute('/_guest/servers')({
    component: Servers,
    beforeLoad: () => {
        ensureFeatureEnabled('servers_enabled');
    }
});

function LinearProgressWithLabel(props: LinearProgressProps & { value: number }) {
    return (
        <Box display="flex" alignItems="center">
            <Box width="100%" mr={1}>
                <LinearProgress variant="determinate" {...props} />
            </Box>
            a
            <Box minWidth={35}>
                <Typography variant="body2" color="textSecondary">{`${Math.round(props.value)}%`}</Typography>
            </Box>
        </Box>
    );
}

export const ServerStats = () => {
    const { servers } = useMapStateCtx();

    const cap = useMemo(() => servers?.length, [servers]);
    const use = useMemo(() => {
        return sum(servers.map((value) => value?.players || 0));
    }, [servers]);

    const regions = useMemo(() => {
        return servers.reduce(
            (acc, cv) => {
                if (!Object.prototype.hasOwnProperty.call(acc, cv.region)) {
                    acc[cv.region] = [];
                }
                acc[cv.region].push(cv);
                return acc;
            },
            {} as Record<string, BaseServer[]>
        );
    }, [servers]);

    const keys = useMemo(() => {
        return Object.keys(regions).sort();
    }, [regions]);

    if (servers.length === 0) {
        return <LoadingPlaceholder />;
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
                <Grid size={{ xs: 3, xl: 4 }}>
                    <Typography style={{ display: 'inline' }} variant={'subtitle1'} align={'center'}>
                        Global: {use} / {cap}
                    </Typography>
                    <LinearProgressWithLabel value={Math.round((use / cap) * 100)} />
                </Grid>

                {keys.map((v) => {
                    const pSum = sum(
                        ((Object.prototype.hasOwnProperty.call(regions, v) && regions[v]) || []).map(
                            (value) => value?.players || 0
                        )
                    );
                    const pMax = sum(
                        ((Object.prototype.hasOwnProperty.call(regions, v) && regions[v]) || []).map(
                            (value) => value?.max_players || 24
                        )
                    );
                    return (
                        <Grid size={{ xs: 3, xl: 4 }} key={`stat-${v}`}>
                            <Typography style={{ display: 'inline' }} variant={'subtitle1'} align={'center'}>
                                {v}: {pSum} / {pMax}
                            </Typography>
                            <LinearProgressWithLabel value={Math.round((pSum / pMax) * 100)} />
                        </Grid>
                    );
                })}
            </Grid>
        </Container>
    );
};

function Servers() {
    const [servers, setServers] = useState<BaseServer[]>([]);
    const { hasPermission } = useAuth();
    const [pos, setPos] = useState<LatLngLiteral>({
        lat: 0.0,
        lng: 0.0
    });
    const [customRange, setCustomRange] = useState<number>(500);
    const [selectedServers, setSelectedServers] = useState<BaseServer[]>([]);
    const [filterByRegion, setFilterByRegion] = useState<boolean>(false);
    const [showOpenOnly, setShowOpenOnly] = useState<boolean>(false);
    const [selectedRegion, setSelectedRegion] = useState<string>('any');
    const [showHelp, setShowHelp] = useState<boolean>(false);

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
        <>
            <Title>Servers</Title>
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
                    {showHelp && <QueueHelp />}
                    <ContainerWithHeaderAndButtons
                        title={`Servers (${selectedServers.length}/${servers.length})`}
                        buttons={
                            !hasPermission(PermissionLevel.Moderator)
                                ? []
                                : [
                                      <Tooltip title={'Toggle server queue help'} key={'help-queue-button'}>
                                          <IconButton
                                              color={'default'}
                                              onClick={() => {
                                                  setShowHelp((prevState) => !prevState);
                                              }}
                                          >
                                              {showHelp ? <HelpIcon /> : <HelpOutlineIcon />}
                                          </IconButton>
                                      </Tooltip>
                                  ]
                        }
                        iconLeft={<StorageIcon />}
                    >
                        <ServerList />
                    </ContainerWithHeaderAndButtons>
                    <ServerStats />
                </Stack>
            </MapStateCtx.Provider>
        </>
    );
}
