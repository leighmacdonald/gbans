import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const DurationCustomFieldValidator = yup.string().required('Duration required').label('Duration');

interface DurationStringFieldProps {
    duration: string;
}

export const DurationStringField = () => {
    const { values, touched, errors, handleChange } = useFormikContext<DurationStringFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                id={'duration'}
                label={'Duration'}
                name={'duration'}
                value={values.duration}
                onChange={handleChange}
                error={touched.duration && Boolean(errors.duration)}
            />
            <FormHelperText>{touched.duration && errors.duration && `${errors.duration}`}</FormHelperText>
        </FormControl>
    );
};
