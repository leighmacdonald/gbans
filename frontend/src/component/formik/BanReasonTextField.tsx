import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { BanReason } from '../../api';
import { BanReasonFieldProps } from './BanReasonField';

export const BanReasonTextFieldValidator = yup
    .string()
    .when('reason', {
        is: BanReason.Custom,
        then: (schema) => schema.required(),
        otherwise: (schema) => schema.notRequired()
    })
    .label('Custom reason');

interface BanReasonTextFieldProps {
    reason_text: BanReason;
}

export const BanReasonTextField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & BanReasonTextFieldProps & BanReasonFieldProps
    >();

    return (
        <TextField
            fullWidth
            id="reason_text"
            name={'reason_text'}
            label="Custom Reason"
            disabled={values.reason != BanReason.Custom}
            value={values.reason_text}
            onChange={handleChange}
            error={touched.reason_text && Boolean(errors.reason_text)}
            helperText={
                touched.reason_text &&
                errors.reason_text &&
                `${errors.reason_text}`
            }
            variant="outlined"
        />
    );
};
