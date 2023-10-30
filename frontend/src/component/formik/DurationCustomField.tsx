import { Duration } from '../../api';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const DurationCustomFieldValidator = yup
    .string()
    .label('Custom duration');

export const DurationCustomField = ({
    formik
}: {
    formik: FormikState<{
        duration: Duration;
        duration_custom: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            label={'Custom Duration'}
            id={'duration_custom'}
            name={'duration_custom'}
            disabled={formik.values.duration != Duration.durCustom}
            value={formik.values.duration_custom}
            onChange={formik.handleChange}
            error={
                formik.touched.duration_custom &&
                Boolean(formik.errors.duration_custom)
            }
            helperText={
                formik.touched.duration_custom && formik.errors.duration_custom
            }
        />
    );
};
