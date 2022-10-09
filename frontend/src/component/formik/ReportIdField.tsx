import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const ReportIdFieldValidator = yup
    .number()
    .min(0, 'Must be positive integer')
    .nullable();

export const ReportIdField = ({
    formik
}: {
    formik: FormikState<{
        reportId?: number;
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
            value={formik.values.reportId}
            onChange={formik.handleChange}
        />
    );
};
