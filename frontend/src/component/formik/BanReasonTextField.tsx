import { useMemo } from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import { BanReason } from '../../api';
import { BanReasonFieldProps } from './BanReasonField';

// const banValidationSchema = yup.object({
//     reason_text: banReasonTextFieldValidator
// });

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
