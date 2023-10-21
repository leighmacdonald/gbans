import React from 'react';
import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
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

export const NetworkRangeField = ({
    formik
}: {
    formik: FormikState<{
        cidr: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            label={'CIDR Network Range'}
            id={'cidr'}
            name={'cidr'}
            value={formik.values.cidr}
            onChange={formik.handleChange}
            error={formik.touched.cidr && Boolean(formik.errors.cidr)}
            helperText={formik.touched.cidr && formik.errors.cidr}
        />
    );
};
