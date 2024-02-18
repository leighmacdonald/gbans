import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface IPFieldProps {
    ip: string;
}

export const IPField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & IPFieldProps
    >();
    return (
        <TextField
            fullWidth
            id="ip"
            name={'ip'}
            label="IP Address"
            value={values.ip}
            onChange={handleChange}
            error={touched.ip && Boolean(errors.ip)}
            helperText={touched.ip && Boolean(errors.ip) && `${errors.ip}`}
            variant="outlined"
        />
    );
};
