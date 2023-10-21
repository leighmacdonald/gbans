import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';

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
