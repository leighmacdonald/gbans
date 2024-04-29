import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface DemoNameFieldProps {
    demo_name: string;
}

export const DemoNameField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<T & DemoNameFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                id={'demo_name'}
                label={'Demo Name'}
                name={'demo_name'}
                value={values.demo_name}
                onChange={handleChange}
                error={touched.demo_name && Boolean(errors.demo_name)}
            />
            <FormHelperText>{touched.demo_name && errors.demo_name && `${errors.demo_name}`}</FormHelperText>
        </FormControl>
    );
};
