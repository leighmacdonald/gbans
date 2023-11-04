import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const NameFieldValidator = yup.string().label('Hidden Moderator Note');

export interface NameFieldProps {
    name: string;
}

export const NameField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & NameFieldProps
    >();
    return (
        <TextField
            fullWidth
            id="name"
            name={'name'}
            label="Optional Name"
            value={values.name}
            onChange={handleChange}
            error={touched.name && Boolean(errors.name)}
            helperText={touched.name && errors.name && `${errors.name}`}
            variant="outlined"
        />
    );
};
