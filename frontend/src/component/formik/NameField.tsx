import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const NameFieldValidator = yup
    .string()
    .min(2, 'Name too short')
    .required('Name required');

export interface NameFieldProps {
    name: string;
}

export const NameField = () => {
    const { values, touched, errors, handleChange } =
        useFormikContext<NameFieldProps>();
    return (
        <TextField
            fullWidth
            id="name"
            name={'name'}
            label="Name"
            value={values.name}
            onChange={handleChange}
            error={touched.name && Boolean(errors.name)}
            helperText={touched.name && errors.name && `${errors.name}`}
            variant="outlined"
        />
    );
};
