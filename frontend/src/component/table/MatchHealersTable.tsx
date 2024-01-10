import React, { useMemo, useState } from 'react';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { MatchHealer, MatchPlayer, Team } from '../../api';
import { blu, red } from '../../theme';
import { PersonCell } from '../PersonCell';
import { LazyTable, Order } from './LazyTable';
import { compare, stableSort } from './LazyTableSimple';

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

export const MatchHealersTable = ({ players }: MatchHealersTableProps) => {
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
