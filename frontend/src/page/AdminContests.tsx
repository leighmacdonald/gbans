import React, { useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';

export const AdminContests = () => {
    return (
        <>
            <Grid container>
                <Grid xs={12}>
                    <Stack>
                        <Button
                            onClick={() => {
                                setEditorOpen(true);
                            }}
                        >
                            Create Contest
                        </Button>
                    </Stack>
                </Grid>
            </Grid>
        </>
    );
};
