import { DatePicker } from '@mui/x-date-pickers';
import { parseISO } from 'date-fns';
import { useFormikContext } from 'formik';

interface DateEndFieldProps {
    date_end: string;
}

export const DateEndField = ({ maxDate }: { maxDate?: Date }) => {
    const { errors, touched, values, setFieldValue } = useFormikContext<DateEndFieldProps>();
    return (
        <DatePicker
            maxDate={maxDate}
            closeOnSelect={true}
            label="Date End"
            value={values.date_end ? parseISO(values.date_end) : ''}
            formatDensity={'dense'}
            onChange={async (value) => {
                await setFieldValue('date_end', value);
            }}
            slotProps={{
                textField: {
                    fullWidth: true,
                    variant: 'outlined',
                    error: touched.date_end && Boolean(errors.date_end)
                }
            }}
        />
    );
};
