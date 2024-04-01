import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export type IPFieldProps = {
    ip: string;
};

export const IPField = () => {
    const { values, touched, errors, handleChange } =
        useFormikContext<IPFieldProps>();
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
