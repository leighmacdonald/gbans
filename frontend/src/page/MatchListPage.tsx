import React from 'react';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { MatchHistory } from '../component/MatchHistory';

export const MatchListPage = (): JSX.Element => {
    return (
        <Stack>
            <Box marginTop={3}>
                <Paper elevation={1}>
                    <Typography variant={'h1'} textAlign={'center'} padding={2}>
                        Match History
                    </Typography>
                </Paper>
            </Box>
            <Box>
                <MatchHistory limit={100} />
            </Box>
        </Stack>
    );
};
