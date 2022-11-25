import React, { useEffect, useState } from 'react';
import { SelectChangeEvent } from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import { apiGetServerStates, ServerState } from '../api';
import { logErr } from '../util/errors';

export interface ServerSelectProps {
    setServerIDs: (servers: number[]) => void;
}

export const ServerSelect = ({ setServerIDs }: ServerSelectProps) => {
    const [servers, setServers] = useState<ServerState[]>();
    const [selectedServers, setSelectedServers] = useState<number[]>([0]);

    useEffect(() => {
        apiGetServerStates()
            .then((servers) => {
                setServers(servers.result || []);
            })
            .catch(logErr);
    }, []);

    const containsAll = (f: number[]): boolean =>
        f.filter((f) => f == 0).length > 0;

    const handleChange = (event: SelectChangeEvent<number[]>) => {
        let newValue: number[];
        const values = event.target.value as number[];
        if (!values || (!containsAll(selectedServers) && containsAll(values))) {
            newValue = [0];
        } else if (values.length > 1) {
            newValue = values.filter((f) => f != 0);
        } else {
            newValue = values;
        }
        setSelectedServers(newValue);
        setServerIDs(newValue);
    };

    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">Servers</InputLabel>
            <Select<number[]>
                labelId="server-select-label"
                multiple
                id="server-select"
                value={selectedServers}
                label="Servers"
                onChange={handleChange}
            >
                <MenuItem value={0}>All</MenuItem>
                {servers &&
                    servers.map((s) => (
                        <MenuItem value={s.server_id} key={s.server_id}>
                            {s.name_short || s.name}
                        </MenuItem>
                    ))}
            </Select>
        </FormControl>
    );
};
