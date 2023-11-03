import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const ASNumberFieldValidator = yup
    .number()
    .required()
    .positive()
    .integer();

interface ASNumberFieldProps {
    as_num: number;
}

export const ASNumberField = <T,>() => {
    const { values, handleChange, touched, errors } = useFormikContext<
        T & ASNumberFieldProps
    >();
    return (
        <TextField
            type={'number'}
            fullWidth
            label={'Autonomous System Number'}
            id={'as_num'}
            name={'as_num'}
            value={values.as_num}
            onChange={handleChange}
            error={touched.as_num && Boolean(errors.as_num)}
            helperText={touched.as_num && errors.as_num && `${errors.as_num}`}
        />
    );
};
