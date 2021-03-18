import React, { useState } from 'react';
import { PlayerBanForm } from '../component/PlayerBanForm';
import { Grid, Paper, Typography } from '@material-ui/core';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { ProfilePanel } from '../component/ProfilePanel';
import { PlayerProfile } from '../util/api';

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    },
    header: {
        paddingBottom: '16px'
    }
}));

export const AdminBan = (): JSX.Element => {
    const classes = useStyles();
    const [profile, setProfile] = useState<PlayerProfile | undefined>();
    return (
        <Grid container spacing={3}>
            <Grid item xs={6}>
                <Paper className={classes.paper}>
                    <Grid item xs={12}>
                        <Typography variant={'h1'}>Ban A Player</Typography>
                    </Grid>
                    <PlayerBanForm
                        onProfileChanged={(p) => {
                            setProfile(p);
                        }}
                    />
                </Paper>
            </Grid>
            <Grid item xs={6}>
                <Paper className={classes.paper}>
                    <Grid item xs={12}>
                        <Typography variant={'h1'}>Player Profile</Typography>
                    </Grid>
                    <ProfilePanel profile={profile} />
                </Paper>
            </Grid>
        </Grid>
    );
};
