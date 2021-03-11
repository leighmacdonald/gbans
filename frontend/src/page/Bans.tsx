import React from "react";
import {makeStyles, Theme} from "@material-ui/core/styles";
import {Grid, Paper} from "@material-ui/core";

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

export const Bans = () => {
    const classes = useStyles();
    return (
        <Grid container spacing={3}>
            <Grid item xs>
                <Paper className={classes.paper}>xdg sdfgsdfg sdfg sdgf sdfg sdg s</Paper>
            </Grid>
            <Grid item xs>
                <Paper className={classes.paper}>xs</Paper>
            </Grid>
            <Grid item xs>
                <Paper className={classes.paper}>xs</Paper>
            </Grid>
        </Grid>

    )
}