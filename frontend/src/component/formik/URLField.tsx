import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const URLFieldValidator = yup
    .string()
    .url('Must be valid url')
    .required('URL required');

export interface URLFieldProps {
    url: string;
}

export const URLField = () => {
    const { values, touched, errors, handleChange } =
        useFormikContext<URLFieldProps>();
    return (
        <TextField
            fullWidth
            id="url"
            name={'url'}
            label="URL"
            value={values.url}
            onChange={handleChange}
            error={touched.url && Boolean(errors.url)}
            helperText={touched.url && errors.url && `${errors.url}`}
            variant="outlined"
        />
    );
};
