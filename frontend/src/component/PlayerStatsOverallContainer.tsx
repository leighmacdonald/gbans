import React, { useEffect, useState, JSX } from 'react';
import { apiGetPlayerStats, PlayerOverallResult } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import Grid from '@mui/material/Unstable_Grid2';
import BarChartIcon from '@mui/icons-material/BarChart';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text';
import { formatDistance } from 'date-fns';
import { fmtWhenGt } from './PlayersOverallContainer';
import Typography from '@mui/material/Typography';

interface PlayerStatsOverallContainerProps {
    steam_id: string;
}

const SimpleStatRow = ({ label, value }: StatPair) => (
    <TableRow hover>
        <TableCell>
            <Typography variant={'button'}>{label}</Typography>
        </TableCell>
        <TableCell>{value}</TableCell>
    </TableRow>
);

interface StatPair {
    label: string;
    value: number | string | JSX.Element;
}

interface BasicStatTableProps {
    stats: StatPair[];
    showHeading?: boolean;
}

const BasicStatTable = ({
    stats,
    showHeading = false
}: BasicStatTableProps) => (
    <TableContainer>
        <Table padding={'none'}>
            {showHeading && (
                <TableHead>
                    <TableRow>
                        <TableCell variant={'head'}>Stat</TableCell>
                        <TableCell variant={'head'}>Value</TableCell>
                    </TableRow>
                </TableHead>
            )}
            <TableBody>
                {stats.map((v) => (
                    <SimpleStatRow key={`stat-${v.label}`} {...v} />
                ))}
            </TableBody>
        </Table>
    </TableContainer>
);

export const PlayerStatsOverallContainer = ({
    steam_id
}: PlayerStatsOverallContainerProps) => {
    const [loading, setLoading] = useState(true);
    const [stats, setStats] = useState<PlayerOverallResult>();

    useEffect(() => {
        apiGetPlayerStats(steam_id)
            .then((resp) => {
                if (resp.result) {
                    setStats(resp.result);
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, [steam_id]);

    return (
        <ContainerWithHeader
            title={'Player Overall Stats'}
            iconLeft={<BarChartIcon />}
        >
            <Grid container spacing={1}>
                <Grid xs={6} md={4}>
                    {loading || !stats ? (
                        <LoadingPlaceholder />
                    ) : (
                        <BasicStatTable
                            stats={[
                                {
                                    label: 'Matches',
                                    value: fmtWhenGt(stats.matches, humanCount)
                                },
                                {
                                    label: 'Wins',
                                    value: fmtWhenGt(stats.wins, humanCount)
                                },
                                {
                                    label: 'Losses',
                                    value: fmtWhenGt(
                                        stats.matches - stats.wins,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Win Rate %',
                                    value: defaultFloatFmtPct(stats.win_rate)
                                },
                                {
                                    label: 'Playtime',
                                    value: `${formatDistance(
                                        0,
                                        stats.playtime * 1000,
                                        {
                                            includeSeconds: false
                                        }
                                    )}`
                                },
                                {
                                    label: 'Extinguishes',
                                    value: fmtWhenGt(
                                        stats.extinguishes,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Buildings Created',
                                    value: fmtWhenGt(
                                        stats.buildings,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Buildings Destroyed',
                                    value: fmtWhenGt(
                                        stats.buildings_destroyed,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Captures',
                                    value: fmtWhenGt(stats.captures, humanCount)
                                },
                                {
                                    label: 'Captured Blocked',
                                    value: fmtWhenGt(
                                        stats.captures_blocked,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Dominations',
                                    value: fmtWhenGt(
                                        stats.dominations,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Dominated',
                                    value: fmtWhenGt(
                                        stats.dominated,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Revenges',
                                    value: fmtWhenGt(stats.revenges, humanCount)
                                }
                            ]}
                        />
                    )}
                </Grid>
                <Grid xs={6} md={4}>
                    {loading || !stats ? (
                        <LoadingPlaceholder />
                    ) : (
                        <BasicStatTable
                            stats={[
                                {
                                    label: 'Kills',
                                    value: fmtWhenGt(stats.kills, humanCount)
                                },
                                {
                                    label: 'Assists',
                                    value: fmtWhenGt(stats.assists, humanCount)
                                },
                                {
                                    label: 'Deaths',
                                    value: fmtWhenGt(stats.deaths, humanCount)
                                },
                                {
                                    label: 'KA',
                                    value: fmtWhenGt(stats.ka, humanCount)
                                },
                                {
                                    label: 'K:D',
                                    value: defaultFloatFmt(stats.kd)
                                },
                                {
                                    label: 'KA:D',
                                    value: defaultFloatFmt(stats.kad)
                                },
                                {
                                    label: 'Damage',
                                    value: fmtWhenGt(stats.damage, humanCount)
                                },
                                {
                                    label: 'Damage Per Min',
                                    value: defaultFloatFmt(stats.dpm)
                                },
                                {
                                    label: 'Shots',
                                    value: fmtWhenGt(stats.shots, humanCount)
                                },
                                {
                                    label: 'Hits',
                                    value: fmtWhenGt(stats.hits, humanCount)
                                },
                                {
                                    label: 'Accuracy',
                                    value: defaultFloatFmtPct(stats.accuracy)
                                },
                                {
                                    label: 'Headshots',
                                    value: fmtWhenGt(
                                        stats.headshots,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Airshots',
                                    value: fmtWhenGt(stats.airshots, humanCount)
                                },
                                {
                                    label: 'Backstabs',
                                    value: fmtWhenGt(
                                        stats.backstabs,
                                        humanCount
                                    )
                                }
                            ]}
                        />
                    )}
                </Grid>
                <Grid xs={6} md={4}>
                    {loading || !stats ? (
                        <LoadingPlaceholder />
                    ) : (
                        <BasicStatTable
                            stats={[
                                {
                                    label: 'Healing',
                                    value: fmtWhenGt(stats.healing, humanCount)
                                },
                                {
                                    label: 'Damage Taken',
                                    value: fmtWhenGt(
                                        stats.damage_taken,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Healing Taken',
                                    value: fmtWhenGt(
                                        stats.healing_taken,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Health Packs',
                                    value: fmtWhenGt(
                                        stats.health_packs,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Drops',
                                    value: fmtWhenGt(stats.drops, humanCount)
                                },
                                {
                                    label: 'Near Full Charge Deaths',
                                    value: fmtWhenGt(
                                        stats.near_full_charge_death,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Avg. Uber Length',
                                    value: defaultFloatFmt(stats.avg_uber_len)
                                },
                                {
                                    label: 'Charges Uber',
                                    value: fmtWhenGt(
                                        stats.charges_uber,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Charges Kritz',
                                    value: fmtWhenGt(
                                        stats.charges_kritz,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Charges Vaccinator',
                                    value: fmtWhenGt(
                                        stats.charges_vacc,
                                        humanCount
                                    )
                                },
                                {
                                    label: 'Charges QuickFix',
                                    value: fmtWhenGt(
                                        stats.charges_quick_fix,
                                        humanCount
                                    )
                                }
                            ]}
                        />
                    )}
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
