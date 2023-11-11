import React from 'react';
import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { emptyOrNullString } from '../../util/types';

export const MapNameFieldValidator = yup
    .string()
    .label('Select a map')
    .min(3, 'Minimum 3 characters required')
    .optional();

export const mapValidator = yup
    .string()
    .test('checkMap', 'Invalid map selection', async (map) => {
        return !emptyOrNullString(map);
    })
    .label('Select a map to play')
    .required('map is required');

interface MapNameFieldProps {
    map_name: string;
}

export const MapNameField = () => {
    const { isSubmitting, values, handleChange, touched, errors } =
        useFormikContext<MapNameFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                disabled={isSubmitting}
                name={'map_name'}
                id={'map_name'}
                label={'Map Name'}
                value={values.map_name}
                onChange={handleChange}
                error={touched.map_name && Boolean(errors.map_name)}
                helperText={touched.map_name && errors.map_name}
            />
        </FormControl>
    );
};
