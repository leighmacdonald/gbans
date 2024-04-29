import { DateTimePicker } from '@mui/x-date-pickers';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { Duration } from '../../api';

export const DurationCustomFieldValidator = yup.date().label('Custom duration');

interface DurationCustomFieldProps {
    duration: Duration;
    duration_custom: Date;
}

export const DurationCustomField = () => {
    const { errors, touched, values, setFieldValue } = useFormikContext<DurationCustomFieldProps>();
    return (
        <DateTimePicker
            disabled={values.duration != Duration.durCustom}
            label="Custom Expiration Date"
            value={values.duration_custom}
            formatDensity={'dense'}
            //onError={(newError) => setError(newError)}
            onChange={async (value) => {
                await setFieldValue('duration_custom', value);
            }}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.duration_custom && Boolean(errors.duration_custom)
                }
            }}
        />
    );
};
