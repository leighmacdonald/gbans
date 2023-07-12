import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';

interface PageNotFoundProps {
    error?: string;
}

export const PageNotFound = ({ error }: PageNotFoundProps) => {
    return (
        <Grid container xs={12} padding={2}>
            <Grid xs={12} alignContent={'center'}>
                <Typography align={'center'} variant={'h1'}>
                    Not Found
                </Typography>
                {error && (
                    <Typography align={'center'} variant={'subtitle1'}>
                        {error}
                    </Typography>
                )}
            </Grid>
        </Grid>
    );
};
