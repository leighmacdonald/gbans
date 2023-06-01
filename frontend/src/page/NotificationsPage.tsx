import React from 'react';
import Grid from '@mui/material/Grid';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { NotificationList } from '../component/NotificationList';

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
