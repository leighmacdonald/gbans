import React, { useMemo, useState } from 'react';
import { Popover } from '@mui/material';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { formatDistance } from 'date-fns';
import { MatchPlayer, MatchPlayerClass, Team } from '../../api';
import { blu, red } from '../../theme';
import { ContainerWithHeader } from '../ContainerWithHeader';
import { PersonCell } from '../PersonCell';
import { PlayerClassImg } from '../PlayerClassImg';
import { PlayerWeaponHoverStats } from '../PlayerWeaponHoverStats';
import { LazyTable, Order } from './LazyTable';
import { compare, stableSort } from './LazyTableSimple';

interface MatchPlayersTableProps {
    players: MatchPlayer[];
}

export const MatchPlayersTable = ({ players }: MatchPlayersTableProps) => {
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

interface PlayerClassHoverStatsProps {
    stats: MatchPlayerClass;
}

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
                    iconLeft={<PlayerClassImg cls={stats.player_class} />}
                    title={'Class Stats'}
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
