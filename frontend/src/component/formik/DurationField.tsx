import React, { useMemo } from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { Duration, Durations, SteamBanRecord } from '../../api';

export const DurationFieldValidator = yup
    .string()
    .label('Ban/Mute duration')
    .required('Duration is required');

export interface DurationInputField {
    duration: Duration;
    existing?: SteamBanRecord;
}

export const DurationField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & DurationInputField
    >();

    const isDisabled = useMemo(() => {
        return values.existing && values.existing.ban_id > 0;
    }, [values.existing]);

    return (
        <FormControl>
            <InputLabel id="duration-label">Duration</InputLabel>
            <Select<Duration>
                fullWidth
                disabled={isDisabled}
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
                        {v != Duration.durInf ? v : 'Permanent'}
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
