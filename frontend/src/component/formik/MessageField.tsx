import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface MessageFieldProps {
    message: string;
}

export const MessageField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & MessageFieldProps
    >();
    return (
        <TextField
            fullWidth
            id="message"
            name={'message'}
            label="Message"
            value={values.message}
            onChange={handleChange}
            error={touched.message && Boolean(errors.message)}
            helperText={
                touched.message && errors.message && `${errors.message}`
            }
            variant="outlined"
        />
    );
};
