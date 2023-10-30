import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import * as yup from 'yup';

export const GroupIdFieldValidator = yup
    .string()
    .length(18, 'Must be positive integer');

export const GroupIdField = ({
    formik
}: {
    formik: FormikState<{
        group_id: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            id="group_id"
            name={'group_id'}
            label="Steam Group ID"
            value={formik.values.group_id}
            onChange={formik.handleChange}
            error={formik.touched.group_id && Boolean(formik.errors.group_id)}
            helperText={formik.touched.group_id && formik.errors.group_id}
            variant="outlined"
        />
    );
};
