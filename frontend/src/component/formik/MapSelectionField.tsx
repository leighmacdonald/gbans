import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import React from 'react';
import * as yup from 'yup';
import { emptyOrNullString } from '../../util/types';

export const BanReasonFieldValidator = yup
    .number()
    .label('Select a reason')
    .required('reason is required');

export const baseMaps = [
    'pl_badwater',
    'cp_process_final',
    'workshop/2834196889'
];

export const mapValidator = yup
    .string()
    .test('checkMap', 'Invalid map selection', async (map) => {
        return !emptyOrNullString(map);
    })
    .label('Select a map to play')
    .required('map is required');

export const MapSelectionField = ({
    formik
}: {
    formik: FormikState<{
        map_name: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="map-selection-label">Map Selection</InputLabel>
            <Select<string>
                disabled={formik.isSubmitting}
                labelId="map-selection-label"
                id="map_name"
                name={'map_name'}
                value={formik.values.map_name}
                onChange={formik.handleChange}
                error={
                    formik.touched.map_name && Boolean(formik.errors.map_name)
                }
            >
                {baseMaps.map((v) => (
                    <MenuItem key={`map_name-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.map_name && formik.errors.map_name}
            </FormHelperText>
        </FormControl>
    );
};
