import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';

export const AdminImportPage = () => {
    return (
        <Grid container>
            <Grid xs>
                <Typography variant={'h1'}>
                    Import Bans & Block Lists
                </Typography>
            </Grid>
        </Grid>
    );
};
