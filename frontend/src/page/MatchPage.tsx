import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import { apiGetMatch, Match } from '../api';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { logErr } from '../util/errors';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';

export const MatchPage = (): JSX.Element => {
    const [playersSortKey, setPlayersSortKey] = useState<string>('profile');
    const [sortDir, setSortDir] = useState<'desc' | 'asc'>('desc');
    const [match, setMatch] = useState<Match>();
    const navigate = useNavigate();
    const { match_id } = useParams();
    const match_id_num = parseInt(match_id || 'x');
    if (isNaN(match_id_num) || match_id_num <= 0) {
        navigate('/404');
    }

    useEffect(() => {
        if (!match) {
            return;
        }
        match.PlayerSums = (match?.PlayerSums || []).sort((a, b): number => {
            switch (playersSortKey) {
                case 'profile':
                    return a.SteamId < b.SteamId
                        ? -1
                        : a.SteamId > b.SteamId
                        ? 1
                        : 0;
                case 'damage':
                    return a.Damage < b.Damage
                        ? -1
                        : a.Damage > b.Damage
                        ? 1
                        : 0;
                case 'damage_taken':
                    return a.DamageTaken < b.DamageTaken
                        ? -1
                        : a.DamageTaken > b.DamageTaken
                        ? 1
                        : 0;
                case 'assists':
                    return a.Assists < b.Assists
                        ? -1
                        : a.Assists > b.Assists
                        ? 1
                        : 0;
                case 'healing':
                    return a.Healing < b.Healing
                        ? -1
                        : a.Healing > b.Healing
                        ? 1
                        : 0;
                case 'healing_taken':
                    return a.HealingTaken < b.HealingTaken
                        ? -1
                        : a.HealingTaken > b.HealingTaken
                        ? 1
                        : 0;
                case 'airshots':
                    return a.Airshots < b.Airshots
                        ? -1
                        : a.Airshots > b.Airshots
                        ? 1
                        : 0;
                case 'headshots':
                    return a.HeadShots < b.HeadShots
                        ? -1
                        : a.HeadShots > b.HeadShots
                        ? 1
                        : 0;
                case 'backstabs':
                    return a.BackStabs < b.BackStabs
                        ? -1
                        : a.BackStabs > b.BackStabs
                        ? 1
                        : 0;
                case 'kills':
                default:
                    return a.Kills < b.Kills ? -1 : a.Kills > b.Kills ? 1 : 0;
            }
        });

        console.log(`sorted ${playersSortKey} ${sortDir}`);
        if (sortDir == 'asc') {
            match.PlayerSums = match?.PlayerSums.reverse();
        }
    }, [playersSortKey, sortDir, setSortDir, match?.PlayerSums, match]);

    useEffect(() => {
        if (match_id_num > 0) {
            apiGetMatch(match_id_num)
                .then((resp) => {
                    setMatch(resp);
                })
                .catch(logErr);
        }
    }, [match_id_num, setMatch]);

    const upd = useCallback(
        (key: string) => {
            setPlayersSortKey(key);
            setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
        },
        [sortDir]
    );

    const mkTitle = (text: string, key: string): JSX.Element => {
        return (
            <Typography
                variant={'button'}
                sx={{
                    padding: 1,
                    '&:hover': {
                        cursor: 'pointer'
                    }
                }}
                onClick={() => {
                    upd(key);
                }}
            >
                {text}
            </Typography>
        );
    };

    return (
        <>
            <Box marginTop={3}>
                <Paper elevation={1}>
                    <Typography
                        variant={'h1'}
                        textAlign={'center'}
                        marginBottom={2}
                    >
                        Match Logs
                    </Typography>
                    <Typography variant={'h4'} textAlign={'center'}>
                        {match?.Title} - {match?.MapName}
                    </Typography>
                </Paper>
            </Box>

            <Grid container spacing={3} paddingTop={3}>
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <Stack>
                            <Grid container>
                                <Grid item xs={2}>
                                    {mkTitle('Profile', 'profile')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Kills', 'kills')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Assists', 'assists')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Deaths', 'deaths')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Damage', 'damage')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('DTaken', 'damage_taken')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Healing', 'healing')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('HTaken', 'healing_taken')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('AS', 'airshots')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('HS', 'headshots')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('BS', 'backstabs')}
                                </Grid>
                            </Grid>
                            {match?.PlayerSums.map((ps) => {
                                return (
                                    <Grid container key={ps.MatchPlayerSumID}>
                                        <Grid item xs={2}>
                                            <Typography
                                                sx={{
                                                    textDecoration: 'none',
                                                    padding: 1
                                                }}
                                                variant={'button'}
                                                component={Link}
                                                to={`/profile/${ps.SteamId}`}
                                            >
                                                {ps.SteamId}
                                            </Typography>
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Kills}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Assists}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Deaths}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Damage}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.DamageTaken}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Healing}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.HealingTaken}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.Airshots}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.HeadShots}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ps.BackStabs}
                                        </Grid>
                                    </Grid>
                                );
                            })}
                        </Stack>
                    </Paper>
                </Grid>

                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <Stack spacing={1}>
                            <Grid container>
                                <Grid item xs={2}>
                                    {mkTitle('Profile', 'profile')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Healing', 'healing')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Charges', 'charges')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Drops', 'drops')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('AvgBuild', 'avg_build')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('NearFull', 'near_full')}
                                </Grid>
                            </Grid>
                            {match?.MedicSums.map((ms) => {
                                return (
                                    <Grid container key={ms.MatchMedicId}>
                                        <Grid item xs={2}>
                                            <Typography
                                                sx={{
                                                    textDecoration: 'none',
                                                    paddingLeft: 1
                                                }}
                                                variant={'button'}
                                                component={Link}
                                                to={`/profile/${ms.SteamId}`}
                                            >
                                                {ms.SteamId}
                                            </Typography>
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ms.Healing}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ms.Charges[0]}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ms.Drops}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ms.AvgTimeToBuild}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ms.NearFullChargeDeath}
                                        </Grid>
                                    </Grid>
                                );
                            })}
                        </Stack>
                    </Paper>
                </Grid>

                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <Stack spacing={1}>
                            <Grid container>
                                <Grid item xs={1}>
                                    {mkTitle('Team', 'team')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Kills', 'kills')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Caps', 'caps')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Mid Fights', 'mid_fights')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Charges', 'charges')}
                                </Grid>
                                <Grid item xs={1}>
                                    {mkTitle('Drops', 'drops')}
                                </Grid>
                            </Grid>
                            {match?.TeamSums.map((ts) => {
                                return (
                                    <Grid container key={ts.MatchTeamId}>
                                        <Grid item xs={1}>
                                            {ts.Team === 1 ? 'RED' : 'BLU'}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ts.Kills}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ts.Caps}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ts.MidFights}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ts.Charges}
                                        </Grid>
                                        <Grid item xs={1}>
                                            {ts.Drops}
                                        </Grid>
                                    </Grid>
                                );
                            })}
                        </Stack>
                    </Paper>
                </Grid>
            </Grid>
        </>
    );
};
