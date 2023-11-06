import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { nonResolvingSteamIDInputTest } from './AuthorIdField';

export const targetIdValidator = yup
    .string()
    .label('Target Steam ID')
    .test(
        'checkTargetId',
        'Invalid target steamid',
        nonResolvingSteamIDInputTest
    );

interface TargetIDInputValue {
    target_id: string;
}

export const TargetIDField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & TargetIDInputValue
    >();
    return (
        <TextField
            fullWidth
            name={'target_id'}
            id={'target_id'}
            label={'Target Steam ID'}
            value={values.target_id}
            onChange={handleChange}
            error={touched.target_id && Boolean(errors.target_id)}
            helperText={
                touched.target_id && errors.target_id && `${errors.target_id}`
            }
        />
    );
};
