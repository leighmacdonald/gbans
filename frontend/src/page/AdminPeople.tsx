import React from 'react';
import { Grid, Paper, Typography } from '@material-ui/core';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { PlayerList } from '../component/PlayerList';

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

export const AdminPeople = (): JSX.Element => {
    const classes = useStyles();
    return (
        <Grid container spacing={3}>
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <Typography variant={'h2'} className={classes.header}>
                        Known Players
                    </Typography>
                    <PlayerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
