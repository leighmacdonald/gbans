import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface CIDRInputFieldProps {
    cidr: string;
}

export const NetworkRangeField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<T & CIDRInputFieldProps>();
    return (
        <TextField
            fullWidth
            label={'CIDR Network Range'}
            id={'cidr'}
            name={'cidr'}
            value={values.cidr}
            onChange={handleChange}
            error={touched.cidr && Boolean(errors.cidr)}
            helperText={touched.cidr && errors.cidr && `${errors.cidr}`}
        />
    );
};
