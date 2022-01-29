import { useEffect, useState } from 'react';
import { apiGetServers, Server } from '../api';
import { SelectChangeEvent } from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import { FormHelperText, InputLabel, Select } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import * as React from 'react';

export const ServerSelect = () => {
    const [servers, setServers] = useState<Server[]>();
    const [selectedServers, setSelectedServers] = useState<string[]>(['']);
    useEffect(() => {
        const f = async () => {
            setServers(await apiGetServers());
        };
        f();
    }, []);

    const containsAll = (f: string[]): boolean => {
        return f.filter((f) => f == '').length > 0;
    };

    const handleChange = (event: SelectChangeEvent<string[]>) => {
        const values = event.target.value as string[];
        if (!values || (!containsAll(selectedServers) && containsAll(values))) {
            setSelectedServers(['']);
            return;
        } else if (values.length > 1) {
            setSelectedServers(values.filter((f) => f != ''));
        } else {
            setSelectedServers(values);
        }
    };

    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">Servers</InputLabel>
            <Select<string[]>
                labelId="server-select-label"
                multiple
                id="server-select"
                value={selectedServers}
                label="Age"
                onChange={handleChange}
            >
                <MenuItem value={''}>All</MenuItem>
                {servers &&
                    servers.map((s) => (
                        <MenuItem value={s.server_name} key={s.server_id}>
                            {s.server_name}
                        </MenuItem>
                    ))}
            </Select>
            <FormHelperText id="server-helper-text">
                Filter events by server id
            </FormHelperText>
        </FormControl>
    );
};
