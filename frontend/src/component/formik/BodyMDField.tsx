import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const bodyMDValidator = yup
    .string()
    .min(2)
    .label('Message Body')
    .required('Message required');

interface BodyMDFieldProps {
    body_md: string;
}

export const BodyMDField = <T,>() => {
    const { handleBlur, values, touched, errors, handleChange } =
        useFormikContext<T & BodyMDFieldProps>();
    return (
        <TextField
            fullWidth
            multiline
            minRows={15}
            id="body_md"
            name={'body_md'}
            label="Message"
            value={values.body_md}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.body_md && Boolean(errors.body_md)}
            helperText={
                touched.body_md && errors.body_md && `${errors.body_md}`
            }
            variant="outlined"
        />
    );
};
