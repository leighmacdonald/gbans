import Grid from '@mui/material/Grid';
import React, { useState } from 'react';
import { Button, TextField } from '@mui/material';

export const ServerAddForm = (): JSX.Element => {
    const [name, setName] = useState<string>('');
    const [address, setAddress] = useState<string>('');
    const [port, setPort] = useState<number>(27015);

    return (
        <Grid container>
            <form noValidate autoComplete="off">
                <Grid item xs={12}>
                    <TextField
                        id="standard-basic"
                        label="Short Server Name (Example: eg-1)"
                        fullWidth
                        value={name}
                        onChange={(v) => {
                            setName(v.target.value);
                        }}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        id="standard-basic"
                        label="Hostname or IP"
                        fullWidth
                        value={address}
                        onChange={(v: any) => {
                            setAddress(v.target.value);
                        }}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        type="number"
                        id="standard-basic"
                        label="Port (Default: 27015)"
                        fullWidth
                        value={port}
                        onChange={(v: any) => {
                            setPort(parseInt(v.target.value));
                        }}
                    />
                </Grid>
                <Grid item xs={12}>
                    <Button variant={'contained'} type={'submit'}>
                        Add Server
                    </Button>
                </Grid>
            </form>
        </Grid>
    );
};
