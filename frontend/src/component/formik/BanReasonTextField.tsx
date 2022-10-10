import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';
import { BanReason } from '../../api';
import * as yup from 'yup';

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
        reasonText: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            id="reasonText"
            name={'reasonText'}
            label="Custom Reason"
            disabled={formik.values.reason != BanReason.Custom}
            value={formik.values.reasonText}
            onChange={formik.handleChange}
            error={
                formik.touched.reasonText && Boolean(formik.errors.reasonText)
            }
            helperText={formik.touched.reasonText && formik.errors.reasonText}
            variant="outlined"
        />
    );
};
