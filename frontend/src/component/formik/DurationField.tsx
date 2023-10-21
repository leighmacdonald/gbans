import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';
import { Duration, Durations } from '../../api';

export const DurationFieldValidator = yup
    .string()
    .label('Ban/Mute duration')
    .required('Duration is required');

export const DurationField = ({
    formik
}: {
    formik: FormikState<{
        duration: Duration;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="duration-label">Duration</InputLabel>
            <Select<Duration>
                fullWidth
                label={'Ban Duration'}
                labelId="duration-label"
                id="duration"
                name={'duration'}
                value={formik.values.duration}
                onChange={formik.handleChange}
                error={
                    formik.touched.duration && Boolean(formik.errors.duration)
                }
            >
                {Durations.map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.duration && formik.errors.duration}
            </FormHelperText>
        </FormControl>
    );
};
