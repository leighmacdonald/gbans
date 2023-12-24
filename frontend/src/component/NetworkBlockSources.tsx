import React from 'react';
import LibraryAddIcon from '@mui/icons-material/LibraryAdd';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Unstable_Grid2';

export const NetworkBlockSources = () => {
    return (
        <Grid container>
            <Grid xs={12}>
                <ButtonGroup>
                    <Button startIcon={<LibraryAddIcon />}>Add New</Button>
                </ButtonGroup>
            </Grid>
            <Grid xs={12}></Grid>
        </Grid>
    );
};
