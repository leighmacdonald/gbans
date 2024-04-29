import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface WeightFieldProps {
    weight: number;
}

export const WeightField = () => {
    const { errors, touched, values, handleBlur, handleChange, isSubmitting } = useFormikContext<WeightFieldProps>();
    return (
        <TextField
            fullWidth
            disabled={isSubmitting}
            name={'weight'}
            id={'weight'}
            label={'Weight'}
            type={'number'}
            value={values.weight}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.weight && Boolean(errors.weight)}
            helperText={touched.weight && errors.weight}
        />
    );
};
