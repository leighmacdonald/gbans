import React, { useEffect, useMemo, useState } from 'react';
import {
    ApiException,
    apiGetMatch,
    Match,
    MatchMedicSum,
    MatchPlayerSum,
    MatchTeamSum
} from '../api';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { UserTableProps, UserTable } from '../component/UserTable';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { first } from 'lodash-es';
import { LoadingSpinner } from '../component/LoadingSpinner';

export const MatchPage = (): JSX.Element => {
    const navigate = useNavigate();
    const [match, setMatch] = useState<Match>();
    const [loading, setLoading] = React.useState<boolean>(true);
    const { match_id } = useParams();
    const match_id_num = parseInt(match_id || 'x');
    const { sendFlash } = useUserFlashCtx();

    if (isNaN(match_id_num) || match_id_num <= 0) {
        sendFlash('error', 'Invalid match id');
        navigate('/404');
    }

    useEffect(() => {
        if (match_id_num > 0) {
            apiGetMatch(match_id_num)
                .then((resp) => {
                    setMatch(resp);
                })
                .catch((r: ApiException) => {
                    sendFlash(
                        'error',
                        r.resp.status == 404
                            ? 'Unknown match id'
                            : 'Internal error'
                    );
                    navigate('/404');
                    return;
                })
                .finally(() => {
                    setLoading(false);
                });
        }
    }, [match_id_num, navigate, sendFlash, setMatch]);

    const playerTableDef: UserTableProps<MatchPlayerSum> = useMemo(() => {
        return {
            columnOrder: [
                'SteamId',
                'Kills',
                'Assists',
                'Deaths',
                'Damage',
                'KDRatio',
                'KADRatio',
                'DamageTaken',
                'Healing',
                'HealingTaken',
                'Airshots',
                'HeadShots',
                'BackStabs'
            ],
            defaultSortColumn: 'Kills',
            columns: [
                {
                    label: 'Profile',
                    sortKey: 'SteamId',
                    tooltip: 'Profile',
                    align: 'left',
                    renderer: (value) => {
                        const p = first(
                            (match?.Players || []).filter(
                                (p) => p.steam_id == value
                            )
                        );
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/profile/${p?.steam_id}`}
                            >
                                {p?.personaname ?? `${p?.steam_id}`}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'K',
                    sortKey: 'Kills',
                    tooltip: 'Kills'
                },
                {
                    label: 'A',
                    sortKey: 'Assists',
                    tooltip: 'Assists'
                },
                {
                    label: 'D',
                    sortKey: 'Deaths',
                    tooltip: 'Deaths'
                },
                {
                    label: 'Dmg',
                    sortKey: 'Damage',
                    tooltip: 'Damage'
                },
                {
                    label: 'KD',
                    sortKey: 'KDRatio',
                    tooltip: 'Kills:Death ratio',
                    sortType: 'float'
                },
                {
                    label: 'KAD',
                    sortKey: 'KADRatio',
                    tooltip: 'Kills+Assists:Death ratio',
                    sortType: 'float'
                },
                {
                    label: 'DT',
                    sortKey: 'DamageTaken',
                    tooltip: 'DTaken'
                },
                {
                    label: 'Healing',
                    sortKey: 'Healing',
                    tooltip: 'Healing'
                },
                {
                    label: 'HT',
                    sortKey: 'HealingTaken',
                    tooltip: 'Healing Taken'
                },
                {
                    label: 'AS',
                    sortKey: 'Airshots',
                    tooltip: 'Airshots'
                },
                {
                    label: 'HS',
                    sortKey: 'HeadShots',
                    tooltip: 'Headshots'
                },
                {
                    label: 'BS',
                    sortKey: 'BackStabs',
                    tooltip: 'Backstabs'
                }
            ],
            rows: match?.PlayerSums || []
        };
    }, [match?.PlayerSums, match?.Players]);

    const medicTableDef: UserTableProps<MatchMedicSum> = useMemo(() => {
        return {
            columnOrder: [
                'SteamId',
                'Healing',
                'Charges',
                'Drops',
                'NearFullChargeDeath'
            ],
            defaultSortColumn: 'Healing',
            columns: [
                {
                    label: 'Profile',
                    sortKey: 'SteamId',
                    tooltip: 'Profile',
                    align: 'left',
                    renderer: (value) => {
                        const p = first(
                            (match?.Players || []).filter(
                                (p) => p.steam_id == value
                            )
                        );
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/profile/${p?.steam_id}`}
                            >
                                {p?.personaname ?? `${p?.steam_id}`}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Healing',
                    sortKey: 'Healing',
                    tooltip: 'Healing'
                },
                {
                    label: 'Charges',
                    sortKey: 'Charges',
                    tooltip: 'Uber Charges',
                    renderer: (value: { K: number } | unknown) => {
                        return value
                            ? Object.values(value as { K: number }).reduce(
                                  (sum, current) => sum + current,
                                  0
                              )
                            : 0;
                    }
                },
                {
                    label: 'Drops',
                    sortKey: 'Drops',
                    tooltip: 'Uber Drops'
                },
                {
                    label: 'Near Full Death',
                    sortKey: 'NearFullChargeDeath',
                    tooltip: 'Near full uber death'
                }
            ],
            rows: match?.MedicSums || []
        };
    }, [match?.MedicSums, match?.Players]);

    const teamTableDef: UserTableProps<MatchTeamSum> = useMemo(() => {
        return {
            columnOrder: [
                'Team',
                'Kills',
                'Damage',
                'Caps',
                'Charges',
                'Drops',
                'MidFights'
            ],
            defaultSortColumn: 'Kills',
            columns: [
                {
                    label: 'Team',
                    sortKey: 'Team',
                    tooltip: 'Team',
                    align: 'left'
                },
                {
                    label: 'Kills',
                    sortKey: 'Kills',
                    tooltip: 'Total Kills'
                },
                {
                    label: 'Damage',
                    sortKey: 'Damage',
                    tooltip: 'Total Team Damage'
                },
                {
                    label: 'Caps',
                    sortKey: 'Caps',
                    tooltip: 'Total Captures'
                },
                {
                    label: 'Ubers',
                    sortKey: 'Charges',
                    tooltip: 'Total uber charges'
                },
                {
                    label: 'Drops',
                    sortKey: 'Drops',
                    tooltip: 'Total uber drops'
                },
                {
                    label: 'Mid',
                    sortKey: 'MidFights',
                    tooltip: 'Midfight wins'
                }
            ],
            rows: match?.TeamSums || []
        };
    }, [match?.TeamSums]);

    return (
        <Stack spacing={3} marginTop={3}>
            {!loading && match?.MatchID && (
                <>
                    <Typography
                        variant={'h1'}
                        textAlign={'center'}
                        marginBottom={2}
                    >
                        Logs #{match?.MatchID} - {match?.Title} -{' '}
                        {match?.MapName}
                    </Typography>
                    <UserTable {...playerTableDef} />
                    <UserTable {...medicTableDef} />
                    <UserTable {...teamTableDef} />
                </>
            )}
            {loading && <LoadingSpinner />}
        </Stack>
    );
};
