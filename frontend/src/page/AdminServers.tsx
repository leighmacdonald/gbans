import React from "react";
import {Grid} from "@material-ui/core";
import {ServerAddForm} from "../component/ServerAddForm";
import {ServerList} from "../component/ServerList";

export const AdminServers = () => {
    return (
        <Grid container>
            <Grid item xs={8}>
                <ServerList />
            </Grid>
            <Grid item xs={4}>
                <ServerAddForm />
            </Grid>
        </Grid>
    )
}