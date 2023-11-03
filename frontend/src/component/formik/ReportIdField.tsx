import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const ReportIdFieldValidator = yup
    .number()
    .min(0, 'Must be positive integer')
    .nullable();

interface ReportIdFieldProps {
    report_id: number;
}

export const ReportIdField = () => {
    const { values, handleChange } = useFormikContext<ReportIdFieldProps>();
    return (
        <TextField
            sx={{ display: 'none' }}
            fullWidth
            id={'report_id'}
            label={'report_id'}
            name={'report_id'}
            disabled={true}
            hidden={true}
            value={values.report_id}
            onChange={handleChange}
        />
    );
};
