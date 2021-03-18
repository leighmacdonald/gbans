import React from 'react';
import { Grid, Paper } from '@material-ui/core';
import { ServerList } from '../component/ServerList';
import { makeStyles, Theme } from '@material-ui/core/styles';

const useStyles = makeStyles((theme: Theme) => ({
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary
    }
}));

export const Servers = (): JSX.Element => {
    const classes = useStyles();
    return (
        <Grid container>
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
