import React, { ChangeEvent, useState } from 'react';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Stack from '@mui/material/Stack';
import { apiCreateServer } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export const ServerAddForm = (): JSX.Element => {
    const [name, setName] = useState<string>('');
    const [address, setAddress] = useState<string>('');
    const [rcon, setRcon] = useState<string>('');
    const [port, setPort] = useState<number>(27015);
    const [lat, setLat] = useState<string>('');
    const [lon, setLon] = useState<string>('');
    const [defaultMap, setDefaultMap] = useState<string>('');
    const [region, setRegion] = useState<string>('');
    const [cc, setCC] = useState<string>('');
    const [reservedSlots, setReservedSlots] = useState<number>(0);
    const { flashes, setFlashes } = useUserFlashCtx();

    const reset = () => {
        setName('');
        setPort(27015);
        setAddress('');
        setRcon('');
        setLat('');
        setLon('');
        setDefaultMap('');
        setRegion('');
        setCC('');
        setReservedSlots(0);
    };

    const addServer = async () => {
        if (name && address && rcon && port > 0) {
            const resp = await apiCreateServer({
                name_short: name,
                host: address,
                port: port,
                rcon: rcon,
                reserved_slots: reservedSlots,
                lat: parseFloat(lat),
                lon: parseFloat(lon),
                cc: cc,
                default_map: defaultMap,
                region: region
            });
            setFlashes([
                ...flashes,
                {
                    closable: true,
                    heading: 'header',
                    level: 'success',
                    message: `Server Creation Successful, password: ${resp.password}`
                }
            ]);
            reset();
        }
    };

    return (
        <Stack spacing={3} padding={2}>
            <Heading>Add a New Server</Heading>
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
                onChange={(v: ChangeEvent<HTMLInputElement>) => {
                    setAddress(v.target.value);
                }}
            />
            <TextField
                type="number"
                id="port"
                label="Port (Default: 27015)"
                fullWidth
                value={port}
                onChange={(v: ChangeEvent<HTMLInputElement>) => {
                    setPort(parseInt(v.target.value));
                }}
            />
            <Stack direction={'row'} spacing={2}>
                <TextField
                    id="region"
                    label="Region"
                    fullWidth
                    value={region}
                    onChange={(v: ChangeEvent<HTMLInputElement>) => {
                        setRegion(v.target.value);
                    }}
                />
                <TextField
                    id="cc"
                    label="Country Code (2 chars)"
                    fullWidth
                    value={cc}
                    onChange={(v: ChangeEvent<HTMLInputElement>) => {
                        const value = v.target.value as string;
                        if (value.length <= 2) {
                            setCC(value);
                        }
                    }}
                />
            </Stack>
            <Stack direction={'row'} spacing={2}>
                <TextField
                    id="lat"
                    label="Latitude"
                    fullWidth
                    value={lat}
                    onChange={(v: ChangeEvent<HTMLInputElement>) => {
                        setLat(v.target.value);
                    }}
                />

                <TextField
                    id="lat"
                    label="Longitude"
                    fullWidth
                    value={lon}
                    onChange={(v: ChangeEvent<HTMLInputElement>) => {
                        setLon(v.target.value);
                    }}
                />
            </Stack>

            <TextField
                type="number"
                id="reserved_slots"
                label="Reserved Slots"
                fullWidth
                value={reservedSlots}
                onChange={(v: ChangeEvent<HTMLInputElement>) => {
                    setReservedSlots(parseInt(v.target.value));
                }}
            />
            <TextField
                id="default_map"
                label="Default Map"
                fullWidth
                value={defaultMap}
                onChange={(v: ChangeEvent<HTMLInputElement>) => {
                    const value = v.target.value as string;
                    setDefaultMap(value);
                }}
            />
            <TextField
                id="rcon"
                label="RCON Password"
                fullWidth
                value={rcon}
                onChange={(v: ChangeEvent<HTMLInputElement>) => {
                    setRcon(v.target.value);
                }}
            />
            <Button variant={'contained'} type={'submit'} onClick={addServer}>
                Add Server
            </Button>
        </Stack>
    );
};
