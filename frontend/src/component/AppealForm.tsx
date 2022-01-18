import React from 'react';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';

export const AppealForm = (): JSX.Element => {
    return (
        <Paper>
            <Grid container spacing={3}>
                <Grid item xs={6}>
                    <Typography variant={'h2'}>
                        Ban Appeal Application
                    </Typography>
                    <TextField
                        fullWidth
                        label="Appeal"
                        id="appeal_body"
                        minRows={10}
                    />
                </Grid>
                <Grid item xs={6}>
                    <Typography variant={'h3'}>Help</Typography>

                    <Typography variant={'body1'}>
                        Once your appeal is posted, your appeal will be reviewed
                        by an Uncletopia moderator.
                    </Typography>
                    <Typography variant={'body1'}>
                        Hostile or inappropriate messages will be ignored, and
                        you may be subject to removal from the Discord server as
                        well.
                    </Typography>
                    <Typography variant={'body1'}>
                        If your appeal involves trying to blame other people who
                        reported you, or other &quot;whataboutisms&quot;,
                        rethink your approach.
                    </Typography>
                    <Typography variant={'body1'}>
                        Appeals that we deem are argued in bad faith will also
                        be permanently banned.
                    </Typography>
                </Grid>
            </Grid>
        </Paper>
    );
};
