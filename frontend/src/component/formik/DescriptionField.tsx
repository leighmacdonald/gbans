import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import React from 'react';
import * as yup from 'yup';

export const descriptionValidator = yup
    .string()
    .label('Description of the game')
    .optional();

export const DescriptionField = ({
    formik
}: {
    formik: FormikState<{
        description: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                disabled={formik.isSubmitting}
                id={'description'}
                label={'description'}
                name={'description'}
                multiline={true}
                rows={10}
                value={formik.values.description}
                onChange={formik.handleChange}
            />
            <FormHelperText>
                {formik.touched.description && formik.errors.description}
            </FormHelperText>
        </FormControl>
    );
};
