import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { UserTable, UserTableProps } from './UserTable';
import { LoadingSpinner } from './LoadingSpinner';
import React, { useEffect, useMemo, useState } from 'react';
import { apiGetMatches, MatchesQueryOpts, MatchSummary } from '../api';
import { Link } from 'react-router-dom';

export const MatchHistory = (opts: MatchesQueryOpts) => {
    const [loading, setLoading] = useState<boolean>(true);
    const [matches, setMatches] = useState<MatchSummary[]>([]);

    useEffect(() => {
        apiGetMatches(opts)
            .then((matches) => {
                setMatches(matches || []);
            })
            .finally(() => {
                setLoading(false);
            });
    }, []);

    const matchTableDef: UserTableProps<MatchSummary> = useMemo(() => {
        return {
            columnOrder: [
                'match_id',
                // 'server_id',
                'map_name',
                'kills',
                'damage',
                'healing',
                'airshots',
                'created_on'
            ],
            defaultSortColumn: 'match_id',
            columns: [
                {
                    tooltip: 'Match ID',
                    label: 'Match ID',
                    sortKey: 'match_id',
                    sortType: 'number',
                    align: 'left',
                    renderer: (value) => {
                        return (
                            <Typography
                                variant={'button'}
                                component={Link}
                                to={`/log/${value}`}
                            >
                                {value as number}
                            </Typography>
                        );
                    }
                },
                {
                    tooltip: 'Map',
                    label: 'Map',
                    sortKey: 'map_name',
                    sortType: 'string'
                },
                {
                    tooltip: 'Kills',
                    label: 'Kills',
                    sortKey: 'kills'
                },
                {
                    tooltip: 'Damage',
                    label: 'Damage',
                    sortKey: 'damage'
                },
                {
                    tooltip: 'Healing',
                    label: 'Healing',
                    sortKey: 'healing'
                },
                {
                    tooltip: 'Airshots',
                    label: 'Airshots',
                    sortKey: 'airshots'
                },
                {
                    tooltip: 'Created',
                    label: 'Created',
                    sortKey: 'created_on',
                    sortType: 'date'
                }
            ],
            rows: matches
        };
    }, [matches]);

    if (loading) {
        return <LoadingSpinner />;
    }
    return (
        <Stack spacing={3} marginTop={3}>
            {matches.length == 0 && (
                <Typography variant={'caption'}>No results</Typography>
            )}
            {matches.length > 0 && <UserTable {...matchTableDef} />}
        </Stack>
    );
};
