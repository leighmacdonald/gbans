import React from 'react';
import { DatePicker } from '@mui/x-date-pickers';
import { useFormikContext } from 'formik';
import { BaseFormikInputProps } from './SteamIdField';

interface DateEndFieldProps {
    date_end: Date;
}

export const DateEndField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, touched, values, setFieldValue } =
        useFormikContext<DateEndFieldProps>();
    return (
        <DatePicker
            disabled={isReadOnly ?? false}
            label="Date End"
            value={values.date_end}
            formatDensity={'dense'}
            //onError={(newError) => setError(newError)}
            onChange={async (value) => {
                await setFieldValue('date_end', value);
            }}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.date_end && Boolean(errors.date_end)
                }
            }}
        />
    );
};
