import React from 'react';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { MatchHistory } from '../component/MatchHistory';
import { Heading } from '../component/Heading';

export const MatchListPage = (): JSX.Element => {
    return (
        <Stack marginTop={3}>
            <Paper>
                <Box>
                    <Heading>Match History</Heading>
                    <MatchHistory opts={{ limit: 100 }} />
                </Box>
            </Paper>
        </Stack>
    );
};
