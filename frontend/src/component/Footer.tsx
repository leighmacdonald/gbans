import React from "react";
import {Grid, Paper, Typography} from "@material-ui/core";

export const Footer = () => {
    return (
        <Grid container spacing={3} alignItems="center" justify="space-evenly">
            <Grid  item xs={6}>
                <Paper>
                    <Typography align={"center"} variant={"body2"}>
                        Powered By: <a href="https://github.com/leighmacdonald/gbans">gbans</a>
                    </Typography>
                </Paper>
            </Grid>
        </Grid>

    )
}