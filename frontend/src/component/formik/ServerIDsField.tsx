import React, { useEffect, useState } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { apiGetServerStates, BaseServer } from '../../api';
import { logErr } from '../../util/errors';

export const serverIDsValidator = yup.array().label('Select a server');

interface ServerSelectFieldProps {
    server_ids: number;
}

export const ServerIDsField = () => {
    const [servers, setServers] = useState<BaseServer[]>();
    const { values, handleChange, touched, errors } =
        useFormikContext<ServerSelectFieldProps>();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetServerStates(abortController)
            .then((servers) => {
                setServers(servers?.servers || []);
            })
            .catch(logErr);
        return () => abortController.abort();
    }, []);

    // const containsAll = (f: number[]): boolean =>
    //     f.filter((f) => f == 0).length > 0;
    //
    // const handleChange = (event: SelectChangeEvent<number[]>) => {
    //     let newValue: number[];
    //     const values = event.target.value as number[];
    //     if (!values || (!containsAll(selectedServers) && containsAll(values))) {
    //         newValue = [0];
    //     } else if (values.length > 1) {
    //         newValue = values.filter((f) => f != 0);
    //     } else {
    //         newValue = values;
    //     }
    //     setSelectedServers(newValue);
    //     setServerIDs(newValue);
    // };

    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">Servers</InputLabel>
            <Select
                labelId="server_ids-label"
                multiple
                id="server_ids"
                value={values.server_ids}
                name={'server_ids'}
                label="Servers"
                error={touched.server_ids && Boolean(errors.server_ids)}
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
