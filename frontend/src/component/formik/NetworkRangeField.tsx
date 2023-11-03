import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const NetworkRangeFieldValidator = yup
    .string()
    .label('Input a CIDR network range')
    .required('CIDR address is required')
    .test('rangeValid', 'Range invalid', (addr) => {
        if (!addr) {
            return false;
        }
        if (!addr.includes('/')) {
            addr = addr + '/32';
        } else {
            const v = addr.split('/');
            if (v.length > 1 && parseInt(v[1]) < 24) {
                return false;
            }
        }
        return true;
    });

export interface CIDRInputFieldProps {
    cidr: string;
}

export const NetworkRangeField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & CIDRInputFieldProps
    >();
    return (
        <TextField
            fullWidth
            label={'CIDR Network Range'}
            id={'cidr'}
            name={'cidr'}
            value={values.cidr}
            onChange={handleChange}
            error={touched.cidr && Boolean(errors.cidr)}
            helperText={touched.cidr && errors.cidr && `${errors.cidr}`}
        />
    );
};
