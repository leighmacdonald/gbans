import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export const serverValidator = yup
    .string()
    .test('checkServerName', 'Invalid server selection', (value) => {
        return demoServers.includes(value as string);
    })
    .label('Select a server to play')
    .required('server is required');

const demoServers = ['sea-1', 'sea-2', 'lax-1', 'chi-1', 'nyc-1'];

export const ServerSelectionField = ({
    formik
}: {
    formik: FormikState<{
        server_name: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="server_name-label">Server Selection</InputLabel>
            <Select<string>
                disabled={formik.isSubmitting}
                labelId="server_name-label"
                id="server_name"
                name={'server_name'}
                value={formik.values.server_name}
                onChange={formik.handleChange}
                error={
                    formik.touched.server_name &&
                    Boolean(formik.errors.server_name)
                }
            >
                {demoServers.map((v) => (
                    <MenuItem key={`server_name-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.server_name && formik.errors.server_name}
            </FormHelperText>
        </FormControl>
    );
};
