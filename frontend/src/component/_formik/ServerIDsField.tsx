import { useEffect, useState } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import { apiGetServerStates, BaseServer } from '../../api';
import { logErr } from '../../util/errors';

interface ServerSelectFieldProps {
    server_ids: number;
}

export const ServerIDsField = () => {
    const [servers, setServers] = useState<BaseServer[]>();
    const { values, handleChange, touched, errors } = useFormikContext<ServerSelectFieldProps>();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetServerStates(abortController)
            .then((servers) => {
                setServers(servers?.servers || []);
            })
            .catch(logErr);
        return () => abortController.abort();
    }, []);

    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">Servers</InputLabel>
            <Select
                fullWidth
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
