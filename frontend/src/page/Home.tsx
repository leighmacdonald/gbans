import React from "react";
import {StatsPanel} from "../component/StatsPanel";
import {BanList} from "../component/BanList";
import {Grid, Paper} from "@material-ui/core";

import {makeStyles, Theme} from "@material-ui/core/styles";
const useStyles = makeStyles((theme: Theme) => ({
    root: {
        flexGrow: 1,
    },
    paper: {
        padding: theme.spacing(2),
        textAlign: 'center',
        color: theme.palette.text.secondary,
    },
}));

export const Home = () => {
    const classes = useStyles();
    return (
        <Grid container spacing={3}>
            <Grid item xs={9}>
                <Paper className={classes.paper}>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs={3}>
                <Paper className={classes.paper}><StatsPanel /></Paper>
            </Grid>
        </Grid>
    )
}