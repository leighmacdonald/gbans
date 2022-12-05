import React from 'react';
import Grid from '@mui/material/Grid';
import { ContainerWithHeader } from './ContainerWithHeader';
import { NotificationList } from './NotificationList';

export const NotificationsPage = () => {
    return (
        <Grid container>
            <Grid item xs={10}>
                <NotificationList />
            </Grid>
            <Grid item xs={2}>
                <ContainerWithHeader title={'Manage'}></ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
