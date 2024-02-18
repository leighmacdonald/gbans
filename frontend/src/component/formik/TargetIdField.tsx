import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface TargetIDInputValue {
    target_id: string;
}

export const TargetIDField = <T,>({
    label = 'Target Steam ID'
}: {
    label?: string;
}) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & TargetIDInputValue
    >();
    return (
        <TextField
            fullWidth
            name={'target_id'}
            id={'target_id'}
            label={label}
            value={values.target_id}
            onChange={handleChange}
            error={touched.target_id && Boolean(errors.target_id)}
            helperText={
                touched.target_id && errors.target_id && `${errors.target_id}`
            }
        />
    );
};
