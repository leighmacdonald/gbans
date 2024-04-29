import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface TitleFieldProps {
    title: string;
}

export const TitleField = () => {
    const { errors, touched, values, handleBlur, handleChange, isSubmitting } = useFormikContext<TitleFieldProps>();
    return (
        <TextField
            fullWidth
            disabled={isSubmitting}
            name={'title'}
            id={'title'}
            label={'Title'}
            value={values.title}
            onChange={handleChange}
            onBlur={handleBlur}
            error={touched.title && Boolean(errors.title)}
            helperText={touched.title && errors.title}
        />
    );
};
