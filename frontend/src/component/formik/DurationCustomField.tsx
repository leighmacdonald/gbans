import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { Duration } from '../../api';
import { DurationInputField } from './DurationField';

export const DurationCustomFieldValidator = yup
    .string()
    .label('Custom duration');

export interface DurationCustomInputField {
    duration_custom: string;
}

export const DurationCustomField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & DurationCustomInputField & DurationInputField
    >();

    return (
        <TextField
            fullWidth
            label={'Custom Duration'}
            id={'duration_custom'}
            name={'duration_custom'}
            disabled={values.duration != Duration.durCustom}
            value={values.duration_custom}
            onChange={handleChange}
            error={touched.duration_custom && Boolean(errors.duration_custom)}
            helperText={
                touched.duration_custom &&
                errors.duration_custom &&
                `${errors.duration_custom}`
            }
        />
    );
};
