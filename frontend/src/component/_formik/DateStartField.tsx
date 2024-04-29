import { DatePicker } from '@mui/x-date-pickers';
import { parseISO } from 'date-fns';
import { useFormikContext } from 'formik';

interface DateStartFieldProps {
    date_start: string;
}

export const DateStartField = ({ minDate }: { minDate?: Date }) => {
    const { errors, setFieldValue, touched, values } = useFormikContext<DateStartFieldProps>();
    return (
        <DatePicker
            minDate={minDate}
            closeOnSelect={true}
            label="Date Start"
            value={values.date_start ? parseISO(values.date_start) : ''}
            onChange={(value) => setFieldValue('date_start', value, true)}
            slotProps={{
                textField: {
                    fullWidth: true,
                    variant: 'outlined',
                    error: touched.date_start && Boolean(errors.date_start)
                }
            }}
        />
    );
};
