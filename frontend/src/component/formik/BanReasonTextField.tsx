import React, { useMemo } from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { BanReason } from '../../api';
import { BanReasonFieldProps } from './BanReasonField';

export const BanReasonTextFieldValidator = yup
    .string()
    .when('reason', {
        is: `${BanReason.Custom}`, // TODO Make BanReason enum work
        then: () =>
            yup.string().required('Custom reason cannot be empty').min(1),
        otherwise: () =>
            yup
                .string()
                .required('Reason cannot be empty')
                .min(1, 'Reason cannot be blank')
    })
    .label('Custom reason');

export const unbanValidationSchema = yup.object({
    reason_text: BanReasonTextFieldValidator
});

interface BanReasonTextFieldProps {
    reason_text: BanReason;
}

export const BanReasonTextField = <T,>({
    paired = true
}: {
    paired?: boolean;
}) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & BanReasonTextFieldProps & BanReasonFieldProps
    >();

    const isError = useMemo(() => {
        if (paired) {
            return (
                values.reason == BanReason.Custom &&
                touched.reason_text &&
                Boolean(errors.reason_text)
            );
        }
        return touched.reason_text && Boolean(errors.reason_text);
    }, [errors.reason_text, paired, touched.reason_text, values.reason]);

    return (
        <TextField
            fullWidth
            id="reason_text"
            name={'reason_text'}
            label="Custom Reason"
            disabled={paired ? values.reason != BanReason.Custom : false}
            value={values.reason_text}
            onChange={handleChange}
            error={isError}
            helperText={
                touched.reason_text &&
                errors.reason_text &&
                `${errors.reason_text}`
            }
            variant="outlined"
        />
    );
};
