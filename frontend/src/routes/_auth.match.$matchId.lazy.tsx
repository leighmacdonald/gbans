import { useMemo, useState, MouseEvent } from 'react';
import GpsFixedIcon from '@mui/icons-material/GpsFixed';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import MasksIcon from '@mui/icons-material/Masks';
import SportsIcon from '@mui/icons-material/Sports';
import { Popover } from '@mui/material';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useQuery } from '@tanstack/react-query';
import { createLazyFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { formatDistance } from 'date-fns';
import { apiGetMatch, MatchHealer, MatchPlayer, MatchPlayerClass, MatchPlayerWeapon, Team } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Heading } from '../component/Heading.tsx';
import { LoadingSpinner } from '../component/LoadingSpinner.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { PlayerClassImg } from '../component/PlayerClassImg.tsx';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import bluLogoImg from '../icons/blu_logo.png';
import redLogoImg from '../icons/red_logo.png';
import { PageNotFound } from './_auth.page-not-found.lazy.tsx';

export const Route = createLazyFileRoute('/_auth/match/$matchId')({
    component: MatchPage
});

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
                <Typography variant={'body2'} padding={1} sx={{ fontFamily: 'Monospace' }}>
                    {value}
                </Typography>
            </TableCell>
        </TableRow>
    );
};

const PlayerClassHoverStats = ({ stats }: PlayerClassHoverStatsProps) => {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

    const handlePopoverOpen = (event: MouseEvent<HTMLElement>) => {
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
                <ContainerWithHeader iconLeft={<PlayerClassImg cls={stats.player_class} />} title={'Class Stats'}>
                    <TableContainer>
                        <Table padding={'none'}>
                            <TableBody>
                                <ClassStatRow name={'Kills'} value={stats.kills} />
                                <ClassStatRow name={'Assists'} value={stats.assists} />
                                <ClassStatRow name={'Deaths'} value={stats.deaths} />
                                <ClassStatRow
                                    name={'Playtime'}
                                    value={formatDistance(0, stats.playtime * 1000, { includeSeconds: true })}
                                />
                                <ClassStatRow name={'Dominations'} value={stats.dominations} />
                                <ClassStatRow name={'Dominated'} value={stats.dominated} />
                                <ClassStatRow name={'Revenges'} value={stats.revenges} />
                                <ClassStatRow name={'Damage'} value={stats.damage} />
                                <ClassStatRow name={'Damage Taken'} value={stats.damage_taken} />
                                <ClassStatRow name={'Healing Taken'} value={stats.healing_taken} />
                                <ClassStatRow name={'Captures'} value={stats.captures} />
                                <ClassStatRow name={'Captures Blocked'} value={stats.captures_blocked} />
                                <ClassStatRow name={'Buildings Destroyed'} value={stats.building_destroyed} />
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

const WeaponCell = ({ value, width }: { value: string | number; width?: number | string }) => {
    return (
        <TableCell width={width ?? 'auto'}>
            <Typography padding={0.5} variant={'body2'} sx={{ fontFamily: 'Monospace' }}>
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
                        ? ((weaponStat.hits / weaponStat.shots) * 100).toFixed(2)
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
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

    const handlePopoverOpen = (event: MouseEvent<HTMLElement>) => {
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
                <ContainerWithHeader title={'Weapon Stats'}>
                    <TableContainer>
                        <Table padding={'checkbox'} size={'small'}>
                            <TableHead>
                                <TableRow>
                                    <TableCell variant="head" width={'400px'}>
                                        <Typography variant={'button'}>Weapon</Typography>
                                    </TableCell>
                                    <TableCell variant="head">
                                        <Typography variant={'button'}>Kills</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>Damage</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>Shots</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>Hits</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>Acc%</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>BS</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>HS</Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>AS</Typography>
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

function MatchPage() {
    const { matchId } = Route.useParams();
    const theme = useTheme();

    const { data: match, isLoading } = useQuery({
        queryKey: ['match', { matchId }],
        queryFn: async () => {
            return await apiGetMatch(matchId);
        }
    });

    const headerColour = useMemo(() => {
        return theme.palette.common.white;
    }, [theme.palette.common.white]);

    if (isLoading) {
        return <LoadingSpinner />;
    }

    if (!match) {
        return <PageNotFound />;
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
                <Grid xs={5} bgcolor={blu} display="flex" justifyContent="left" alignItems="center">
                    <img src={bluLogoImg} alt={'BLU Team'} />
                </Grid>
                <Grid xs={1} bgcolor={blu} display="flex" justifyContent="right" alignItems="center">
                    <Typography variant={'h1'} textAlign={'right'} color={headerColour} sx={{ fontWeight: 900 }}>
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
                    <MatchPlayersTable players={match.players} isLoading={isLoading} />
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <Heading align={'center'} iconLeft={<MasksIcon />}>
                        Healers
                    </Heading>
                </Grid>
                <Grid xs={12} padding={0} paddingTop={1}>
                    <MatchHealersTable players={match.players} isLoading={isLoading} />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
}

const MatchPlayersTable = ({ players, isLoading }: { players: MatchPlayer[]; isLoading: boolean }) => {
    const columnHelper = createColumnHelper<MatchPlayer>();
    const columns = [
        columnHelper.accessor('team', {
            header: () => <TableHeadingCell name={'Team'} />,
            cell: (info) => (
                <Typography color={players[info.row.index].team == Team.BLU ? blu : red} textAlign={'center'}>
                    {info.getValue() == Team.RED ? 'RED' : 'BLU'}
                </Typography>
            )
        }),
        columnHelper.accessor('name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={players[info.row.index].steam_id}
                    personaname={players[info.row.index].name}
                    avatar_hash={players[info.row.index].avatar_hash}
                />
            )
        }),
        columnHelper.accessor('classes', {
            header: () => <TableHeadingCell name={'Classes'} />,
            cell: (info) => (
                <TableCellSmall>
                    {info.getValue() ? (
                        info
                            .getValue()
                            .map((pc) => (
                                <PlayerClassHoverStats
                                    key={`pc-${players[info.row.index].steam_id}-${pc.player_class}`}
                                    stats={pc}
                                />
                            ))
                    ) : (
                        <></>
                    )}
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('weapons', {
            header: () => <TableHeadingCell name={'W'} />,
            cell: (info) => (
                <TableCellSmall>
                    <PlayerWeaponHoverStats stats={info.getValue()} />
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('kills', {
            header: () => <TableHeadingCell name={'K'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('assists', {
            header: () => <TableHeadingCell name={'A'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('deaths', {
            header: () => <TableHeadingCell name={'D'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('damage', {
            header: () => <TableHeadingCell name={'Dmg'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('damage_taken', {
            header: () => <TableHeadingCell name={'DT'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('health_packs', {
            header: () => <TableHeadingCell name={'HP'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('backstabs', {
            header: () => <TableHeadingCell name={'BS'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('headshots', {
            header: () => <TableHeadingCell name={'HS'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('airshots', {
            header: () => <TableHeadingCell name={'AS'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('captures', {
            header: () => <TableHeadingCell name={'CP'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        })
    ];

    const table = useReactTable({
        data: players,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};

interface MedicRow extends MatchHealer {
    steam_id: string;
    team: Team;
    name: string;
    avatar_hash: string;
    time_start: Date;
    time_end: Date;
}

const MatchHealersTable = ({ players, isLoading }: { players: MatchPlayer[]; isLoading: boolean }) => {
    const medics = useMemo(() => {
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
                    near_full_charge_death: p.medic_stats?.near_full_charge_death ?? 0
                };
            });
    }, [players]);

    const columnHelper = createColumnHelper<MedicRow>();
    const columns = [
        columnHelper.accessor('team', {
            header: () => <TableHeadingCell name={'Team'} />,
            cell: (info) => (
                <Typography color={players[info.row.index].team == Team.BLU ? blu : red} textAlign={'center'}>
                    {info.getValue() == Team.RED ? 'RED' : 'BLU'}
                </Typography>
            )
        }),
        columnHelper.accessor('name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={players[info.row.index].steam_id}
                    personaname={players[info.row.index].name}
                    avatar_hash={players[info.row.index].avatar_hash}
                />
            )
        }),

        columnHelper.accessor('healing', {
            header: () => <TableHeadingCell name={'Healing'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),

        columnHelper.accessor('charges_uber', {
            header: () => <TableHeadingCell name={'Uber'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('charges_kritz', {
            header: () => <TableHeadingCell name={'Kritz'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('charges_vacc', {
            header: () => <TableHeadingCell name={'Vacc'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('charges_quickfix', {
            header: () => <TableHeadingCell name={'Quickfix'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('drops', {
            header: () => <TableHeadingCell name={'Drops'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        }),
        columnHelper.accessor('avg_uber_length', {
            header: () => <TableHeadingCell name={'Avg. Len'} />,
            cell: (info) => <TableCellSmall>{info.getValue()}</TableCellSmall>
        })
    ];

    const table = useReactTable({
        data: medics,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
