import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const weightFieldValidator = yup
    .number()
    .min(1, 'Min weight is 1')
    .required('Weight required')
    .label('Weight');

interface WeightFieldProps {
    weight: number;
}

export const WeightField = () => {
    const { errors, touched, values, handleBlur, handleChange, isSubmitting } =
        useFormikContext<WeightFieldProps>();
    return (
        <TextField
            fullWidth
            disabled={isSubmitting}
            name={'weight'}
            id={'weight'}
            label={'Weight'}
            type={'number'}
            value={values.weight}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.weight && Boolean(errors.weight)}
            helperText={touched.weight && errors.weight}
        />
    );
};
