import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import React from 'react';
import * as yup from 'yup';

export const ReportIdFieldValidator = yup
    .number()
    .min(0, 'Must be positive integer')
    .nullable();

export const ReportIdField = ({
    formik
}: {
    formik: FormikState<{
        report_id?: number;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            sx={{ display: 'none' }}
            fullWidth
            id={'report_id'}
            label={'report_id'}
            name={'report_id'}
            disabled={true}
            hidden={true}
            value={formik.values.report_id}
            onChange={formik.handleChange}
        />
    );
};
