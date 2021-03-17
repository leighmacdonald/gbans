import React, { useState } from 'react';
import { PlayerBanForm } from '../component/PlayerBanForm';
import { Grid, Paper } from '@material-ui/core';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { ProfilePanel } from '../component/ProfilePanel';
import { PlayerProfile } from '../util/api';

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    }
}));

export const AdminBan = (): JSX.Element => {
    const classes = useStyles();
    const [profile, setProfile] = useState<PlayerProfile | undefined>();
    return (
        <Grid container spacing={3}>
            <Grid item xs={6}>
                <Paper className={classes.paper}>
                    <PlayerBanForm
                        onProfileChanged={(p) => {
                            setProfile(p);
                        }}
                    />
                </Paper>
            </Grid>
            <Grid item xs={6}>
                <Paper className={classes.paper}>
                    <ProfilePanel profile={profile} />
                </Paper>
            </Grid>
        </Grid>
    );
};
