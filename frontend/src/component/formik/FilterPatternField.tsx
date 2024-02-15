import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface FilterPatternFieldProps {
    pattern: string;
}

export const FilterPatternField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & FilterPatternFieldProps
    >();
    return (
        <TextField
            fullWidth
            id="note"
            name={'pattern'}
            label="Filter Pattern"
            multiline
            value={values.pattern}
            onChange={handleChange}
            error={touched.pattern && Boolean(errors.pattern)}
            helperText={
                touched.pattern && errors.pattern && `${errors.pattern}`
            }
            variant="outlined"
        />
    );
};
