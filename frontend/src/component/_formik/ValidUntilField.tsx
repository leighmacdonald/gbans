import { DatePicker } from '@mui/x-date-pickers';
import { useFormikContext } from 'formik';
import { BaseFormikInputProps } from './SteamIdField';

interface ValidUntilFieldProps {
    valid_until: Date;
}

export const ValidUntilField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, setFieldValue } = useFormikContext<ValidUntilFieldProps>();
    return (
        <DatePicker
            disabled={isReadOnly ?? false}
            label="Valid Until"
            value={values.valid_until}
            formatDensity={'dense'}
            //onError={(newError) => setError(newError)}
            onChange={async (value) => {
                await setFieldValue('valid_until', value);
            }}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.valid_until && Boolean(errors.valid_until)
                }
            }}
        />
    );
};
