import React, { useEffect, useMemo, useState } from 'react';
import { apiGetMatch, MatchPlayer, MatchResult, Team } from '../api';
import { useNavigate, useParams } from 'react-router-dom';
import Stack from '@mui/material/Stack';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { logErr } from '../util/errors';
import Grid from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { PageNotFound } from './PageNotFound';
import { LazyTable } from '../component/LazyTable';
import { Order } from '../component/DataTable';
import { PlayerClassImg } from '../component/PlayerClassImg';

export const MatchPage = () => {
    const navigate = useNavigate();
    const [match, setMatch] = useState<MatchResult>();
    const [loading, setLoading] = React.useState<boolean>(true);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof MatchPlayer>('kills');
    const { match_id } = useParams<string>();
    const { sendFlash } = useUserFlashCtx();

    if (!match_id || match_id == '') {
        sendFlash('error', 'Invalid match id');
        navigate('/404');
    }

    useEffect(() => {
        apiGetMatch(match_id as string)
            .then((resp) => {
                if (!resp.status || !resp.result) {
                    //navigate('/404');
                    return;
                }
                setMatch(resp.result);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, [match_id, navigate, sendFlash, setMatch]);

    const validRows = useMemo(() => {
        return match ? match.players.filter((m) => m.classes != null) : [];
    }, [match]);

    if (loading) {
        return <LoadingSpinner />;
    }

    if (!match) {
        return <PageNotFound error={'Unknown match id'} />;
    }

    return (
        <ContainerWithHeader title={'Match Results'}>
            <Grid container spacing={3} paddingTop={3}>
                <Grid xs={6}>
                    <Stack>
                        <Typography variant={'subtitle1'}>
                            {match.title}
                        </Typography>
                        <Typography variant={'subtitle2'}>
                            {match.map_name}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={6}>
                    <Stack>
                        <Typography variant={'subtitle1'} textAlign={'right'}>
                            {match.time_start.toString()}
                        </Typography>
                        <Typography variant={'subtitle2'} textAlign={'right'}>
                            {match.time_end.toString()}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={5}>
                    <Typography variant={'h1'}>BLU</Typography>
                </Grid>
                <Grid xs={1}>
                    <Typography variant={'h1'}>
                        {match.team_scores.blu}
                    </Typography>
                </Grid>
                <Grid xs={1}>
                    <Typography variant={'h1'}>
                        {match.team_scores.red}
                    </Typography>
                </Grid>
                <Grid xs={5}>
                    <Typography variant={'h1'} textAlign={'right'}>
                        RED
                    </Typography>
                </Grid>

                <Grid xs={12}>
                    <LazyTable<MatchPlayer>
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        rows={validRows}
                        columns={[
                            {
                                label: 'Team',
                                tooltip: 'Team',
                                sortKey: 'team',
                                sortable: true,
                                align: 'left',
                                width: 100,
                                renderer: (row) => (
                                    <Typography variant={'button'}>
                                        {row.team == Team.RED ? 'RED' : 'BLU'}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Name',
                                tooltip: 'In Game Name',
                                sortKey: 'name',
                                sortable: true,
                                align: 'left',
                                width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.name}
                                    </Typography>
                                )
                            },
                            {
                                label: 'C',
                                tooltip: 'Classes',
                                sortKey: 'classes',
                                align: 'left',
                                //width: 50,
                                renderer: (row) => (
                                    <Stack direction={'row'}>
                                        {row.classes ? (
                                            row.classes.map((pc) => (
                                                <PlayerClassImg
                                                    key={`pc-${row.steam_id}-${pc.player_class}`}
                                                    cls={pc.player_class}
                                                />
                                            ))
                                        ) : (
                                            <></>
                                        )}
                                    </Stack>
                                )
                            },
                            {
                                label: 'K',
                                tooltip: 'Kills',
                                sortKey: 'kills',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.kills}
                                    </Typography>
                                )
                            },
                            {
                                label: 'A',
                                tooltip: 'Assists',
                                sortKey: 'assists',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.assists}
                                    </Typography>
                                )
                            },
                            {
                                label: 'D',
                                tooltip: 'Deaths',
                                sortKey: 'deaths',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.deaths}
                                    </Typography>
                                )
                            },
                            {
                                label: 'DA',
                                tooltip: 'Damage',
                                sortKey: 'damage',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.damage}
                                    </Typography>
                                )
                            },
                            {
                                label: 'DT',
                                tooltip: 'Damage Taken',
                                sortKey: 'damage_taken',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.damage_taken}
                                    </Typography>
                                )
                            },
                            {
                                label: 'HP',
                                tooltip: 'Health Packs',
                                sortKey: 'health_packs',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.health_packs}
                                    </Typography>
                                )
                            },
                            {
                                label: 'BS',
                                tooltip: 'Backstabs',
                                sortKey: 'backstabs',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.backstabs}
                                    </Typography>
                                )
                            },
                            {
                                label: 'HS',
                                tooltip: 'Headshots',
                                sortKey: 'headshots',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.headshots}
                                    </Typography>
                                )
                            },
                            {
                                label: 'AS',
                                tooltip: 'Airshots',
                                sortKey: 'airshots',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.airshots}
                                    </Typography>
                                )
                            },
                            {
                                label: 'CAP',
                                tooltip: 'Point Captures',
                                sortKey: 'captures',
                                sortable: true,
                                align: 'left',
                                //width: 250,
                                renderer: (row) => (
                                    <Typography variant={'body2'}>
                                        {row.captures}
                                    </Typography>
                                )
                            }
                        ]}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
