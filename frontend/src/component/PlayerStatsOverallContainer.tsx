import { useEffect, useState, JSX } from 'react';
import BarChartIcon from '@mui/icons-material/BarChart';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { formatDistance } from 'date-fns';
import { apiGetPlayerStats, PlayerOverallResult } from '../api';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import FmtWhenGt from './FmtWhenGT.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder';

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

const BasicStatTable = ({ stats, showHeading = false }: BasicStatTableProps) => (
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

export const PlayerStatsOverallContainer = ({ steam_id }: PlayerStatsOverallContainerProps) => {
    const [loading, setLoading] = useState(true);
    const [stats, setStats] = useState<PlayerOverallResult>();

    useEffect(() => {
        const abortController = new AbortController();

        apiGetPlayerStats(steam_id, abortController)
            .then((resp) => {
                setStats(resp);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [steam_id]);

    return (
        <ContainerWithHeader title={'Player Overall Stats'} iconLeft={<BarChartIcon />}>
            <Grid container spacing={1}>
                <Grid xs={6} md={4}>
                    {loading || !stats ? (
                        <LoadingPlaceholder />
                    ) : (
                        <BasicStatTable
                            stats={[
                                {
                                    label: 'Matches',
                                    value: FmtWhenGt(stats.matches, humanCount)
                                },
                                {
                                    label: 'Wins',
                                    value: FmtWhenGt(stats.wins, humanCount)
                                },
                                {
                                    label: 'Losses',
                                    value: FmtWhenGt(stats.matches - stats.wins, humanCount)
                                },
                                {
                                    label: 'Win Rate %',
                                    value: defaultFloatFmtPct(stats.win_rate)
                                },
                                {
                                    label: 'Playtime',
                                    value: `${formatDistance(0, stats.playtime * 1000, {
                                        includeSeconds: false
                                    })}`
                                },
                                {
                                    label: 'Extinguishes',
                                    value: FmtWhenGt(stats.extinguishes, humanCount)
                                },
                                {
                                    label: 'Buildings Created',
                                    value: FmtWhenGt(stats.buildings, humanCount)
                                },
                                {
                                    label: 'Buildings Destroyed',
                                    value: FmtWhenGt(stats.buildings_destroyed, humanCount)
                                },
                                {
                                    label: 'Captures',
                                    value: FmtWhenGt(stats.captures, humanCount)
                                },
                                {
                                    label: 'Captured Blocked',
                                    value: FmtWhenGt(stats.captures_blocked, humanCount)
                                },
                                {
                                    label: 'Dominations',
                                    value: FmtWhenGt(stats.dominations, humanCount)
                                },
                                {
                                    label: 'Dominated',
                                    value: FmtWhenGt(stats.dominated, humanCount)
                                },
                                {
                                    label: 'Revenges',
                                    value: FmtWhenGt(stats.revenges, humanCount)
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
                                    value: FmtWhenGt(stats.kills, humanCount)
                                },
                                {
                                    label: 'Assists',
                                    value: FmtWhenGt(stats.assists, humanCount)
                                },
                                {
                                    label: 'Deaths',
                                    value: FmtWhenGt(stats.deaths, humanCount)
                                },
                                {
                                    label: 'KA',
                                    value: FmtWhenGt(stats.ka, humanCount)
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
                                    value: FmtWhenGt(stats.damage, humanCount)
                                },
                                {
                                    label: 'Damage Per Min',
                                    value: defaultFloatFmt(stats.dpm)
                                },
                                {
                                    label: 'Shots',
                                    value: FmtWhenGt(stats.shots, humanCount)
                                },
                                {
                                    label: 'Hits',
                                    value: FmtWhenGt(stats.hits, humanCount)
                                },
                                {
                                    label: 'Accuracy',
                                    value: defaultFloatFmtPct(stats.accuracy)
                                },
                                {
                                    label: 'Headshots',
                                    value: FmtWhenGt(stats.headshots, humanCount)
                                },
                                {
                                    label: 'Airshots',
                                    value: FmtWhenGt(stats.airshots, humanCount)
                                },
                                {
                                    label: 'Backstabs',
                                    value: FmtWhenGt(stats.backstabs, humanCount)
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
                                    value: FmtWhenGt(stats.healing, humanCount)
                                },
                                {
                                    label: 'Damage Taken',
                                    value: FmtWhenGt(stats.damage_taken, humanCount)
                                },
                                {
                                    label: 'Healing Taken',
                                    value: FmtWhenGt(stats.healing_taken, humanCount)
                                },
                                {
                                    label: 'Health Packs',
                                    value: FmtWhenGt(stats.health_packs, humanCount)
                                },
                                {
                                    label: 'Drops',
                                    value: FmtWhenGt(stats.drops, humanCount)
                                },
                                {
                                    label: 'Near Full Charge Deaths',
                                    value: FmtWhenGt(stats.near_full_charge_death, humanCount)
                                },
                                {
                                    label: 'Avg. Uber Length',
                                    value: defaultFloatFmt(stats.avg_uber_len)
                                },
                                {
                                    label: 'Charges Uber',
                                    value: FmtWhenGt(stats.charges_uber, humanCount)
                                },
                                {
                                    label: 'Charges Kritz',
                                    value: FmtWhenGt(stats.charges_kritz, humanCount)
                                },
                                {
                                    label: 'Charges Vaccinator',
                                    value: FmtWhenGt(stats.charges_vacc, humanCount)
                                },
                                {
                                    label: 'Charges QuickFix',
                                    value: FmtWhenGt(stats.charges_quickfix, humanCount)
                                }
                            ]}
                        />
                    )}
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
