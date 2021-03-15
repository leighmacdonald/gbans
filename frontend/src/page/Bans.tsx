import React from 'react';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { Grid, Paper } from '@material-ui/core';
import { BanList } from '../component/BanList';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        flexGrow: 1
    },
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    }
}));

export const Bans = (): JSX.Element => {
    const classes = useStyles();
    return (
        <Grid container spacing={3}>
            <Grid item xs>
                <Paper className={classes.paper}>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs>
                <Paper className={classes.paper}>xs</Paper>
            </Grid>
            <Grid item xs>
                <Paper className={classes.paper}>xs</Paper>
            </Grid>
        </Grid>
    );
};
