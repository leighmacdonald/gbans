import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import { BanReason, BanReasons, banReasonsList } from '../../api';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export const BanReasonFieldValidator = yup
    .number()
    .label('Select a reason')
    .required('reason is required');

export const BanReasonField = ({
    formik
}: {
    formik: FormikState<{
        reason: BanReason;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="reason-label">Reason</InputLabel>
            <Select<BanReason>
                labelId="reason-label"
                id="reason"
                name={'reason'}
                value={formik.values.reason}
                onChange={formik.handleChange}
                error={formik.touched.reason && Boolean(formik.errors.reason)}
            >
                {banReasonsList.map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {BanReasons[v]}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.reason && formik.errors.reason}
            </FormHelperText>
        </FormControl>
    );
};
