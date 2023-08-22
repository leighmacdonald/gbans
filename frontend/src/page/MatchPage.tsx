import React, { useEffect, useMemo, useState } from 'react';
import {
    apiGetMatch,
    MatchPlayer,
    MatchPlayerClass,
    MatchResult,
    Team
} from '../api';
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
import { Popover } from '@mui/material';
import TableContainer from '@mui/material/TableContainer';
import TableRow from '@mui/material/TableRow';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import { formatDistance } from 'date-fns';
import SportsIcon from '@mui/icons-material/Sports';

interface PlayerClassHoverStatsProps {
    stats: MatchPlayerClass;
}

interface ClassStatRowProp {
    name: string;
    value: string | number;
}

const ClassStatRow = ({ name, value }: ClassStatRowProp) => {
    return (
        <TableRow>
            <TableCell>
                <Typography variant={'body1'} padding={1}>
                    {name}
                </Typography>
            </TableCell>
            <TableCell>
                <Typography
                    variant={'body2'}
                    padding={1}
                    sx={{ fontFamily: 'Monospace' }}
                >
                    {value}
                </Typography>
            </TableCell>
        </TableRow>
    );
};
const PlayerClassHoverStats = ({ stats }: PlayerClassHoverStatsProps) => {
    const [anchorEl, setAnchorEl] = React.useState<HTMLElement | null>(null);

    const handlePopoverOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handlePopoverClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);

    return (
        <div>
            <PlayerClassImg
                cls={stats.player_class}
                onMouseEnter={handlePopoverOpen}
                onMouseLeave={handlePopoverClose}
            />
            <Popover
                id="mouse-over-popover"
                sx={{
                    pointerEvents: 'none'
                }}
                open={open}
                anchorEl={anchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left'
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
                onClose={handlePopoverClose}
                disableRestoreFocus
            >
                <ContainerWithHeader
                    iconRight={<PlayerClassImg cls={stats.player_class} />}
                    title={'Class Stats'}
                    align={'space-between'}
                >
                    <TableContainer>
                        <Table padding={'none'}>
                            <TableBody>
                                <ClassStatRow
                                    name={'Kills'}
                                    value={stats.kills}
                                />
                                <ClassStatRow
                                    name={'Assists'}
                                    value={stats.assists}
                                />
                                <ClassStatRow
                                    name={'Deaths'}
                                    value={stats.deaths}
                                />
                                <ClassStatRow
                                    name={'Playtime'}
                                    value={formatDistance(
                                        0,
                                        stats.playtime * 1000,
                                        { includeSeconds: true }
                                    )}
                                />
                                <ClassStatRow
                                    name={'Dominations'}
                                    value={stats.dominations}
                                />
                                <ClassStatRow
                                    name={'Dominated'}
                                    value={stats.dominated}
                                />
                                <ClassStatRow
                                    name={'Revenges'}
                                    value={stats.revenges}
                                />
                                <ClassStatRow
                                    name={'Damage'}
                                    value={stats.damage}
                                />
                                <ClassStatRow
                                    name={'Damage Taken'}
                                    value={stats.damage_taken}
                                />
                                <ClassStatRow
                                    name={'Healing Taken'}
                                    value={stats.healing_taken}
                                />
                                <ClassStatRow
                                    name={'Captures'}
                                    value={stats.captures}
                                />
                                <ClassStatRow
                                    name={'Captures Blocked'}
                                    value={stats.captures_blocked}
                                />
                                <ClassStatRow
                                    name={'Buildings Destroyed'}
                                    value={stats.building_destroyed}
                                />
                            </TableBody>
                        </Table>
                    </TableContainer>
                </ContainerWithHeader>
            </Popover>
        </div>
    );
};
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
    const blu = '#547d8c';
    const red = '#a7584b';
    return (
        <ContainerWithHeader title={'Match Results'} iconLeft={<SportsIcon />}>
            <Grid container spacing={2}>
                <Grid xs={8}>
                    <Stack>
                        <Typography variant={'h1'}>{match.title}</Typography>
                    </Stack>
                </Grid>
                <Grid xs={4}>
                    <Stack>
                        <Typography variant={'h6'} textAlign={'right'}>
                            {match.map_name}
                        </Typography>
                        <Typography variant={'h6'} textAlign={'right'}>
                            {formatDistance(match.time_start, match.time_end, {
                                includeSeconds: true
                            })}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={5} bgcolor={blu}>
                    <Typography variant={'h1'} sx={{ fontWeight: 700 }}>
                        BLU
                    </Typography>
                </Grid>
                <Grid xs={1} bgcolor={blu}>
                    <Typography variant={'h1'} textAlign={'right'}>
                        {match.team_scores.blu}
                    </Typography>
                </Grid>
                <Grid xs={1} bgcolor={red}>
                    <Typography variant={'h1'}>
                        {match.team_scores.red}
                    </Typography>
                </Grid>
                <Grid xs={5} bgcolor={red}>
                    <Typography
                        variant={'h1'}
                        textAlign={'right'}
                        sx={{ fontWeight: 700 }}
                    >
                        RED
                    </Typography>
                </Grid>

                <Grid xs={12} padding={0} paddingTop={1}>
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
                                    <Typography
                                        variant={'button'}
                                        sx={{
                                            backgroundColor:
                                                row.team == Team.BLU ? blu : red
                                        }}
                                    >
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
                                        {row.name != ''
                                            ? row.name
                                            : row.steam_id}
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
                                                <PlayerClassHoverStats
                                                    key={`pc-${row.steam_id}-${pc.player_class}`}
                                                    stats={pc}
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
                                    <Typography
                                        variant={'body2'}
                                        sx={{ fontFamily: 'Monospace' }}
                                    >
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
