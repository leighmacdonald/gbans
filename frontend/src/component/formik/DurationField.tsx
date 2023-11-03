import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { Duration, Durations } from '../../api';

export const DurationFieldValidator = yup
    .string()
    .label('Ban/Mute duration')
    .required('Duration is required');

export interface DurationInputField {
    duration: Duration;
}

export const DurationField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & DurationInputField
    >();
    return (
        <FormControl fullWidth>
            <InputLabel id="duration-label">Duration</InputLabel>
            <Select<Duration>
                fullWidth
                label={'Ban Duration'}
                labelId="duration-label"
                id="duration"
                name={'duration'}
                value={values.duration}
                onChange={handleChange}
                error={touched.duration && Boolean(errors.duration)}
            >
                {Durations.map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.duration &&
                    errors.duration &&
                    errors.duration.toString()}
            </FormHelperText>
        </FormControl>
    );
};
