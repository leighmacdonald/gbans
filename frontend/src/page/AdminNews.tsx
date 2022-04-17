import React from 'react';
import Grid from '@mui/material/Grid';
import { NewsEditorForm } from '../component/NewsEditorForm';
import Paper from '@mui/material/Paper';
import { NewsList } from '../component/NewsList';

export const AdminNews = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={8}>
                <Paper elevation={1}>
                    <NewsEditorForm />
                </Paper>
            </Grid>
            <Grid item xs={4}>
                <Paper elevation={1}>
                    <NewsList />
                </Paper>
            </Grid>
        </Grid>
    );
};
