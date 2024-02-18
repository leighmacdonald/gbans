import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import { BanReasonFieldProps } from './BanReasonField';

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
