import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { BanReasonFieldProps } from './BanReasonField';

export const unbanReasonTextFieldValidator = yup
    .string()
    .min(5, 'Message to short')
    .label('Unban Reason')
    .required('Reason is required');

export const unbanValidationSchema = yup.object({
    unban_reason: unbanReasonTextFieldValidator
});

interface BanReasonTextFieldProps {
    unban_reason: string;
}

export const UnbanReasonTextField = <T,>() => {
    const { values, isSubmitting, touched, errors, handleChange } =
        useFormikContext<T & BanReasonTextFieldProps & BanReasonFieldProps>();

    return (
        <TextField
            fullWidth
            id="unban_reason"
            name={'unban_reason'}
            label="Unban Reason"
            disabled={isSubmitting}
            value={values.unban_reason}
            onChange={handleChange}
            error={touched.unban_reason && Boolean(errors.unban_reason)}
            helperText={
                touched.unban_reason &&
                errors.unban_reason &&
                `${errors.unban_reason}`
            }
            variant="outlined"
        />
    );
};
