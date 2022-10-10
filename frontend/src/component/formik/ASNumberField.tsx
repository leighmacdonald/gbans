import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const ASNumberFieldValidator = yup
    .number()
    .required()
    .positive()
    .integer();

export const ASNumberField = ({
    formik
}: {
    formik: FormikState<{
        asNum: number;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            type={'number'}
            fullWidth
            label={'Autonomous System Number'}
            id={'asNum'}
            name={'asNum'}
            value={formik.values.asNum}
            onChange={formik.handleChange}
            error={formik.touched.asNum && Boolean(formik.errors.asNum)}
            helperText={formik.touched.asNum && formik.errors.asNum}
        />
    );
};
