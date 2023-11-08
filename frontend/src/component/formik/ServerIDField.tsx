import React from 'react';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import stc from 'string-to-color';
import { ServerSimple } from '../../api';

interface ServerIDFieldProps {
    server_id: number;
}

export const ServerIDField = <T,>({ servers }: { servers: ServerSimple[] }) => {
    const { values, handleChange } = useFormikContext<T & ServerIDFieldProps>();
    return (
        <Select<number>
            fullWidth
            value={values.server_id}
            name={'server_id'}
            id={'server_id'}
            onChange={handleChange}
            label={'Server'}
        >
            {servers.map((server) => {
                return (
                    <MenuItem value={server.server_id} key={server.server_id}>
                        {server.server_name}
                    </MenuItem>
                );
            })}
        </Select>
    );
};

export const ServerIDCell = ({
    server_id,
    server_name
}: {
    server_id: number;
    server_name: string;
}) => {
    const { setFieldValue, submitForm } =
        useFormikContext<ServerIDFieldProps>();

    return (
        <Button
            fullWidth
            variant={'text'}
            sx={{
                color: stc(server_name)
            }}
            onClick={async () => {
                await setFieldValue('server_id', server_id);
                await submitForm();
            }}
        >
            {server_name}
        </Button>
    );
};
