import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import React from 'react';
import * as yup from 'yup';
import { BanReason } from '../../api';

export const BanReasonTextFieldValidator = yup
    .string()
    .when('reason', {
        is: BanReason.Custom,
        then: (schema) => schema.required(),
        otherwise: (schema) => schema.notRequired()
    })
    .label('Custom reason');

export const BanReasonTextField = ({
    formik
}: {
    formik: FormikState<{
        reason: BanReason;
        reason_text: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            id="reason_text"
            name={'reason_text'}
            label="Custom Reason"
            disabled={formik.values.reason != BanReason.Custom}
            value={formik.values.reason_text}
            onChange={formik.handleChange}
            error={
                formik.touched.reason_text && Boolean(formik.errors.reason_text)
            }
            helperText={formik.touched.reason_text && formik.errors.reason_text}
            variant="outlined"
        />
    );
};
