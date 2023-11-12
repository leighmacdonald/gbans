import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { isValidIP } from '../../util/text';
import { emptyOrNullString } from '../../util/types';

export const ipFieldValidator = yup
    .string()
    .test('valid_ip', 'Invalid IP', (value) => {
        if (emptyOrNullString(value)) {
            return true;
        }
        return isValidIP(value as string);
    });

interface IPFieldProps {
    ip: string;
}

export const IPField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & IPFieldProps
    >();
    return (
        <TextField
            fullWidth
            id="ip"
            name={'ip'}
            label="IP Address"
            value={values.ip}
            onChange={handleChange}
            error={touched.ip && Boolean(errors.ip)}
            helperText={touched.ip && Boolean(errors.ip) && `${errors.ip}`}
            variant="outlined"
        />
    );
};
