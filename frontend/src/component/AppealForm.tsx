import React from 'react';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';

export const AppealForm = (): JSX.Element => {
    return (
        <>
            <Typography variant={'h2'}>Ban Appeal Application</Typography>
            <TextField fullWidth label="Appeal" id="appeal_body" minRows={10} />
        </>
    );
};
