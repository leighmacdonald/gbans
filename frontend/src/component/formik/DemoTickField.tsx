import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface DemoNameFieldProps {
    demo_tick: string;
}

export const DemTickField = <T,>() => {
    const { isSubmitting, values, touched, errors, handleChange } =
        useFormikContext<T & DemoNameFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                type={'number'}
                disabled={isSubmitting}
                id={'demo_tick'}
                label={'Demo Tick'}
                name={'demo_tick'}
                value={values.demo_tick}
                onChange={handleChange}
                error={touched.demo_tick && Boolean(errors.demo_tick)}
            />
            <FormHelperText>
                {touched.demo_tick && errors.demo_tick && `${errors.demo_tick}`}
            </FormHelperText>
        </FormControl>
    );
};
