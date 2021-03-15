import React from 'react';
import { Grid } from '@material-ui/core';
import { ServerList } from '../component/ServerList';

export const Servers = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item>
                <ServerList />
            </Grid>
        </Grid>
    );
};
