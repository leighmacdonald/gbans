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
        durationCustom: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            label={'Custom Duration'}
            id={'durationCustom'}
            name={'durationCustom'}
            disabled={formik.values.duration != Duration.durCustom}
            value={formik.values.durationCustom}
            onChange={formik.handleChange}
            error={
                formik.touched.durationCustom &&
                Boolean(formik.errors.durationCustom)
            }
            helperText={
                formik.touched.durationCustom && formik.errors.durationCustom
            }
        />
    );
};
