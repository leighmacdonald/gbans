import React, { JSX } from 'react';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { Heading } from '../component/Heading';

export const MatchListPage = (): JSX.Element => {
    return (
        <Stack marginTop={3}>
            <Paper>
                <Box>
                    <Heading>Match History</Heading>
                </Box>
            </Paper>
        </Stack>
    );
};
