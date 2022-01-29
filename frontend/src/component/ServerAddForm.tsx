import React, { useState } from 'react';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';

export const ServerAddForm = (): JSX.Element => {
    const [name, setName] = useState<string>('');
    const [address, setAddress] = useState<string>('');
    const [rcon, setRcon] = useState<string>('');
    const [port, setPort] = useState<number>(27015);

    return (
        <Stack spacing={3} padding={3}>
            <Box color={'primary'}>
                <Typography variant={'h4'}>Add a New Server</Typography>
            </Box>
            <TextField
                id="standard-basic"
                label="Short Server Name (Example: eg-1)"
                fullWidth
                value={name}
                onChange={(v) => {
                    setName(v.target.value);
                }}
            />
            <TextField
                id="address"
                label="Hostname or IP"
                fullWidth
                value={address}
                onChange={(v: any) => {
                    setAddress(v.target.value);
                }}
            />
            <TextField
                type="number"
                id="port"
                label="Port (Default: 27015)"
                fullWidth
                value={port}
                onChange={(v: any) => {
                    setPort(parseInt(v.target.value));
                }}
            />
            <TextField
                id="rcon"
                label="RCON Password"
                fullWidth
                value={rcon}
                onChange={(v: any) => {
                    setRcon(v.target.value);
                }}
            />
            <Button variant={'contained'} type={'submit'}>
                Add Server
            </Button>
        </Stack>
    );
};
