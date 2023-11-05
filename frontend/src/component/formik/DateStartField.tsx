import React from 'react';
import { DatePicker } from '@mui/x-date-pickers';
import { useFormikContext } from 'formik';
import { BaseFormikInputProps } from './SteamIdField';

interface DateStartFieldProps {
    date_start: Date;
}

export const DateStartField = ({ isReadOnly }: BaseFormikInputProps) => {
    const { errors, setFieldValue, touched, values } =
        useFormikContext<DateStartFieldProps>();
    return (
        <DatePicker
            disabled={isReadOnly ?? false}
            label="Date Start"
            value={values.date_start}
            //onError={(newError) => setFieldError('date_end', newError)}
            onChange={(value) => setFieldValue('date_start', value, true)}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: touched.date_start && Boolean(errors.date_start)
                    //helperText: formik.touched.date_end && formik.errors.date_end
                }
            }}
        />
    );
};
