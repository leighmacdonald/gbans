import React from 'react';
import { StatsPanel } from '../component/StatsPanel';
import { BanList } from '../component/BanList';
import { Grid, Paper, Typography } from '@material-ui/core';

import { makeStyles, Theme } from '@material-ui/core/styles';
import { ServerList } from '../component/ServerList';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        flexGrow: 1
    },
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    },
    header: {
        paddingBottom: '16px'
    }
}));

export const Home = (): JSX.Element => {
    const classes = useStyles();
    return (
        <Grid container spacing={3}>
            <Grid item xs={9}>
                <Paper className={classes.paper}>
                    <Typography variant={'h2'} className={classes.header}>
                        Most Recent Bans
                    </Typography>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs={3}>
                <Paper className={classes.paper}>
                    <Typography variant={'h2'} className={classes.header}>
                        DB Stats
                    </Typography>
                    <StatsPanel />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <Typography variant={'h2'} className={classes.header}>
                        Server List
                    </Typography>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
