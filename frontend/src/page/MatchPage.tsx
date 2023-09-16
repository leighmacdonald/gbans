import React, { useEffect, useMemo, useState } from 'react';
import {
    apiGetMatch,
    MatchHealer,
    MatchPlayer,
    MatchPlayerClass,
    MatchPlayerWeapon,
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
import { compare, Order, stableSort } from '../component/DataTable';
import { PlayerClassImg } from '../component/PlayerClassImg';
import { Popover } from '@mui/material';
import TableContainer from '@mui/material/TableContainer';
import TableRow from '@mui/material/TableRow';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import { formatDistance } from 'date-fns';
import SportsIcon from '@mui/icons-material/Sports';
import { Heading } from '../component/Heading';
import { PersonCell } from '../component/PersonCell';
import { useTheme } from '@mui/material/styles';
import MasksIcon from '@mui/icons-material/Masks';
import GpsFixedIcon from '@mui/icons-material/GpsFixed';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import TableHead from '@mui/material/TableHead';
import bluLogoImg from '../icons/blu_logo.png';
import redLogoImg from '../icons/red_logo.png';
import Box from '@mui/material/Box';

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
        <Box display="flex" justifyContent="right" alignItems="center">
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
        </Box>
    );
};
interface WeaponStatRowProps {
    weaponStat: MatchPlayerWeapon;
}

const WeaponCell = ({
    value,
    width
}: {
    value: string | number;
    width?: number | string;
}) => {
    return (
        <TableCell width={width ?? 'auto'}>
            <Typography
                padding={0.5}
                variant={'body2'}
                sx={{ fontFamily: 'Monospace' }}
            >
                {value}
            </Typography>
        </TableCell>
    );
};

const WeaponStatRow = ({ weaponStat }: WeaponStatRowProps) => {
    return (
        <TableRow>
            <WeaponCell value={weaponStat.name} width={'400px'} />
            <WeaponCell value={weaponStat.kills} />
            <WeaponCell value={weaponStat.damage} />
            <WeaponCell value={weaponStat.shots} />
            <WeaponCell value={weaponStat.hits} />
            <WeaponCell
                value={`${
                    !isNaN((weaponStat.hits / weaponStat.shots) * 100)
                        ? ((weaponStat.hits / weaponStat.shots) * 100).toFixed(
                              2
                          )
                        : 0
                }%`}
            />
            <WeaponCell value={weaponStat.backstabs} />
            <WeaponCell value={weaponStat.headshots} />
            <WeaponCell value={weaponStat.airshots} />
        </TableRow>
    );
};

interface PlayerWeaponHoverStatsProps {
    stats: MatchPlayerWeapon[];
}

const PlayerWeaponHoverStats = ({ stats }: PlayerWeaponHoverStatsProps) => {
    const [anchorEl, setAnchorEl] = React.useState<HTMLElement | null>(null);

    const handlePopoverOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handlePopoverClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    return (
        <Box>
            <Box
                display="flex"
                justifyContent="right"
                alignItems="center"
                onMouseEnter={handlePopoverOpen}
                onMouseLeave={handlePopoverClose}
            >
                <InfoOutlinedIcon />
            </Box>
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
                    title={'Weapon Stats'}
                    align={'space-between'}
                >
                    <TableContainer>
                        <Table padding={'checkbox'} size={'small'}>
                            <TableHead>
                                <TableRow>
                                    <TableCell variant="head" width={'400px'}>
                                        <Typography variant={'button'}>
                                            Weapon
                                        </Typography>
                                    </TableCell>
                                    <TableCell variant="head">
                                        <Typography variant={'button'}>
                                            Kills
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Damage
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Shots
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Hits
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Acc%
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            BS
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            HS
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            AS
                                        </Typography>
                                    </TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {stats.map((ws, index) => {
                                    return (
                                        <WeaponStatRow
                                            weaponStat={ws}
                                            key={`ws-${ws.damage}-${ws.weapon_id}-${index}`}
                                        />
                                    );
                                })}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </ContainerWithHeader>
            </Popover>
        </Box>
    );
};

const blu = '#547d8c';
const red = '#a7584b';

export const MatchPage = () => {
    const navigate = useNavigate();
    const [match, setMatch] = useState<MatchResult>();
    const [loading, setLoading] = React.useState<boolean>(true);
    const { match_id } = useParams<string>();
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();

    if (!match_id || match_id == '') {
        sendFlash('error', 'Invalid match id');
        navigate('/404');
    }

    useEffect(() => {
        apiGetMatch(match_id as string)
            .then((resp) => {
                setMatch(resp);
            })
            .catch(logErr)
            .finally(() => {
                setLoading(false);
            });
    }, [match_id, navigate, sendFlash, setMatch]);

    const headerColour = useMemo(() => {
        return theme.palette.common.white;
    }, [theme.palette.common.white]);

    if (loading) {
        return <LoadingSpinner />;
    }

    if (!match) {
        return <PageNotFound error={'Unknown match id'} />;
    }

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
                <Grid
                    xs={5}
                    bgcolor={blu}
                    display="flex"
                    justifyContent="left"
                    alignItems="center"
                >
                    <img src={bluLogoImg} alt={'BLU Team'} />
                </Grid>
                <Grid
                    xs={1}
                    bgcolor={blu}
                    display="flex"
                    justifyContent="right"
                    alignItems="center"
                >
                    <Typography
                        variant={'h1'}
                        textAlign={'right'}
                        color={headerColour}
                        sx={{ fontWeight: 900 }}
                    >
                        {match.team_scores.blu}
                    </Typography>
                </Grid>
                <Grid
                    xs={1}
                    bgcolor={red}
                    color={headerColour}
                    display="flex"
                    justifyContent="left"
                    alignItems="center"
                >
                    <Typography variant={'h1'} sx={{ fontWeight: 900 }}>
                        {match.team_scores.red}
                    </Typography>
                </Grid>
                <Grid
                    xs={5}
                    bgcolor={red}
                    color={headerColour}
                    display="flex"
                    justifyContent="right"
                    alignItems="center"
                >
                    <img src={redLogoImg} alt={'RED Team'} />
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <Heading align={'center'} iconLeft={<GpsFixedIcon />}>
                        Players
                    </Heading>
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <MatchPlayersTable players={match.players} />
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <Heading align={'center'} iconLeft={<MasksIcon />}>
                        Healers
                    </Heading>
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <MatchHealersTable players={match.players} />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};

interface MatchPlayersTableProps {
    players: MatchPlayer[];
}

const MatchPlayersTable = ({ players }: MatchPlayersTableProps) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof MatchPlayer>('kills');
    const theme = useTheme();

    const validRows = useMemo(() => {
        return stableSort(
            players.filter(
                (m) =>
                    m.classes != null &&
                    !(m.kills == 0 && m.assists == 0 && m.deaths == 0)
            ),
            compare(sortOrder, sortColumn)
        );
    }, [players, sortColumn, sortOrder]);

    return (
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
                    align: 'center',
                    width: '50px',
                    style: (row) => {
                        return {
                            height: '100%',
                            backgroundColor: row.team == Team.BLU ? blu : red,
                            textAlign: 'center'
                        };
                    },
                    renderer: (row) => (
                        <Typography
                            variant={'button'}
                            color={theme.palette.common.white}
                        >
                            {row.team == Team.RED ? 'RED' : 'BLU'}
                        </Typography>
                    )
                },
                {
                    label: 'Player Name',
                    tooltip: 'In Game Name',
                    sortKey: 'name',
                    sortable: true,
                    align: 'left',
                    //width: 250,
                    renderer: (row) => (
                        <PersonCell
                            steam_id={row.steam_id}
                            personaname={
                                row.name != '' ? row.name : row.steam_id
                            }
                            avatar_hash={row.avatar_hash}
                        />
                        // <Typography variant={'body1'}>
                        //     {row.name != '' ? row.name : row.steam_id}
                        // </Typography>
                    )
                },
                {
                    label: '',
                    tooltip: 'Classes',
                    sortKey: 'classes',
                    align: 'left',
                    //width: 50,
                    renderer: (row) => (
                        <Stack
                            direction={'row'}
                            display="flex"
                            justifyContent="right"
                            alignItems="center"
                        >
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
                    label: '',
                    tooltip: 'Detailed Weapon Stats',
                    virtual: true,
                    virtualKey: 'weapons',
                    sortable: false,
                    align: 'center',
                    // width: '25px',
                    renderer: (row) => (
                        <PlayerWeaponHoverStats stats={row.weapons} />
                    )
                },
                {
                    label: 'K',
                    tooltip: 'Kills',
                    sortKey: 'kills',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
                    width: '25px',
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
                    width: '25px',
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
                    width: '25px',
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
                    width: '25px',
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
                    tooltip: 'Total Health Packs',
                    sortKey: 'health_packs',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
                    tooltip: 'Total Backstabs',
                    sortKey: 'backstabs',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
                    tooltip: 'Total Headshots',
                    sortKey: 'headshots',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
                    tooltip: 'Total Airshots',
                    sortKey: 'airshots',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
                    tooltip: 'Total Point Captures',
                    sortKey: 'captures',
                    sortable: true,
                    align: 'left',
                    width: '25px',
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
    );
};

interface MatchHealersTableProps {
    players: MatchPlayer[];
}

interface MedicRow extends MatchHealer {
    steam_id: string;
    team: Team;
    name: string;
    avatar_hash: string;
    time_start: Date;
    time_end: Date;
}

const MatchHealersTable = ({ players }: MatchHealersTableProps) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc'),
        [sortColumn, setSortColumn] = useState<keyof MedicRow>('healing'),
        rows = useMemo(() => {
            return players
                .filter((p) => p.medic_stats)
                .map((p): MedicRow => {
                    return {
                        match_player_id: p.match_player_id,
                        steam_id: p.steam_id,
                        avatar_hash: p.avatar_hash,
                        name: p.name,
                        team: p.team,
                        time_start: p.time_start,
                        time_end: p.time_end,
                        healing: p.medic_stats?.healing ?? 0,
                        avg_uber_length: p.medic_stats?.avg_uber_length ?? 0,
                        biggest_adv_lost: p.medic_stats?.biggest_adv_lost ?? 0,
                        charges_kritz: p.medic_stats?.charges_kritz ?? 0,
                        charges_uber: p.medic_stats?.charges_uber ?? 0,
                        charges_vacc: p.medic_stats?.charges_vacc ?? 0,
                        charges_quickfix: p.medic_stats?.charges_quickfix ?? 0,
                        drops: p.medic_stats?.drops ?? 0,
                        match_medic_id: p.medic_stats?.match_medic_id ?? 0,
                        major_adv_lost: p.medic_stats?.major_adv_lost ?? 0,
                        near_full_charge_death:
                            p.medic_stats?.near_full_charge_death ?? 0
                    };
                });
        }, [players]),
        validRows = useMemo(() => {
            return stableSort(rows, compare(sortOrder, sortColumn));
        }, [rows, sortColumn, sortOrder]);

    const theme = useTheme();

    return (
        <LazyTable<MedicRow>
            columns={[
                {
                    label: 'Team',
                    tooltip: 'Team',
                    sortKey: 'team',
                    sortable: true,
                    align: 'center',
                    width: '50px',
                    style: (row) => {
                        return {
                            height: '100%',
                            backgroundColor: row.team == Team.BLU ? blu : red,
                            textAlign: 'center'
                        };
                    },
                    renderer: (row) => (
                        <Typography
                            variant={'button'}
                            sx={{
                                color: theme.palette.common.white
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
                        <PersonCell
                            steam_id={row.steam_id}
                            personaname={
                                row.name != '' ? row.name : row.steam_id
                            }
                            avatar_hash={row.avatar_hash}
                        />
                    )
                },
                {
                    label: 'Healing',
                    tooltip: 'Total healing',
                    sortKey: 'healing',
                    sortable: true,
                    align: 'left',
                    width: 250,
                    renderer: (row) => (
                        <Typography variant={'body1'}>{row.healing}</Typography>
                    )
                },
                {
                    label: 'Uber',
                    tooltip: 'Total Uber Deploys',
                    sortKey: 'charges_uber',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.charges_uber}
                        </Typography>
                    )
                },
                {
                    label: 'Krit',
                    tooltip: 'Total Kritz Deploys',
                    sortKey: 'charges_kritz',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.charges_kritz}
                        </Typography>
                    )
                },
                {
                    label: 'Vacc',
                    tooltip: 'Total Uber Deploys',
                    sortKey: 'charges_vacc',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.charges_vacc}
                        </Typography>
                    )
                },
                {
                    label: 'Quickfix',
                    tooltip: 'Total Uber Deploys',
                    sortKey: 'charges_quickfix',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.charges_quickfix}
                        </Typography>
                    )
                },
                {
                    label: 'Drops',
                    tooltip: 'Total Drops',
                    sortKey: 'drops',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>{row.drops}</Typography>
                    )
                },
                {
                    label: 'Avg. Len',
                    tooltip: 'Average Uber Length',
                    sortKey: 'avg_uber_length',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.avg_uber_length}
                        </Typography>
                    )
                }
            ]}
            sortColumn={sortColumn}
            onSortColumnChanged={async (column) => {
                setSortColumn(column);
            }}
            onSortOrderChanged={async (direction) => {
                setSortOrder(direction);
            }}
            sortOrder={sortOrder}
            rows={validRows}
        />
    );
};
