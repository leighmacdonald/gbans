import React from 'react';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';

export const ReportForm = (): JSX.Element => {
    return (
        <>
            <Typography variant={'h2'}>Report A Player</Typography>
            <TextField fullWidth label="Report Subject" id="report_subject" />
            <TextField
                fullWidth
                label="Description"
                id="report_description"
                minRows={10}
            />
        </>
    );
};
