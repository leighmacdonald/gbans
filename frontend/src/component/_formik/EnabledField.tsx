import Checkbox from '@mui/material/Checkbox';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const EnabledFieldValidator = yup.boolean().required('Enabled required');

export interface EnabledFieldProps {
    enabled: boolean;
}

export const EnabledField = () => {
    const { values, touched, errors, handleChange, isSubmitting } = useFormikContext<EnabledFieldProps>();
    return (
        <FormControl fullWidth>
            <FormGroup>
                <FormControlLabel
                    disabled={isSubmitting}
                    control={<Checkbox name={'enabled'} checked={values.enabled} onChange={handleChange} />}
                    label={'Enabled'}
                />
            </FormGroup>
            <FormHelperText>{touched.enabled && errors.enabled && `${errors.enabled}`}</FormHelperText>
        </FormControl>
    );
};
