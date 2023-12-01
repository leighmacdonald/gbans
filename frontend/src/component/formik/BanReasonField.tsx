import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { BanReason, BanReasons, banReasonsList } from '../../api';

export const banReasonFieldValidator = yup
    .string()
    .label('Select a reason')
    .required('reason is required');

export interface BanReasonFieldProps {
    reason: BanReason;
}

export const BanReasonField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & BanReasonFieldProps
    >();

    return (
        <FormControl fullWidth>
            <InputLabel id="reason-label">Reason</InputLabel>
            <Select<BanReason>
                labelId="reason-label"
                id="reason"
                name={'reason'}
                value={values.reason}
                onChange={handleChange}
                error={touched.reason && Boolean(errors.reason)}
            >
                {banReasonsList.map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {BanReasons[v]}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.reason && errors.reason && `${errors.reason}`}
            </FormHelperText>
        </FormControl>
    );
};
