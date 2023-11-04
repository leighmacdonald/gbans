import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const descriptionValidator = yup
    .string()
    .label('Description of the game')
    .optional();

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
                label={'description'}
                name={'description'}
                multiline={true}
                rows={10}
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
