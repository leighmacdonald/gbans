import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface DescriptionFieldProps {
    description: string;
}

export const DescriptionField = <T,>() => {
    const { isSubmitting, values, touched, errors, handleChange } =
        useFormikContext<T & DescriptionFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                disabled={isSubmitting}
                id={'description'}
                label={'Description'}
                name={'description'}
                multiline={true}
                minRows={10}
                maxRows={20}
                value={values.description}
                onChange={handleChange}
                error={touched.description && Boolean(errors.description)}
            />
            <FormHelperText>
                {touched.description &&
                    errors.description &&
                    `${errors.description}`}
            </FormHelperText>
        </FormControl>
    );
};
