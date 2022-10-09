import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const NetworkRangeFieldValidator = yup
    .string()
    .label('Input a CIDR network range')
    .required('CIDR address is required');

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
