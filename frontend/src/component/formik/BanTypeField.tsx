import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';
import { BanType } from '../../api';

export const BanTypeFieldValidator = yup
    .number()
    .label('Select a ban type')
    .required('ban type is required');

export const BanTypeField = ({
    formik
}: {
    formik: FormikState<{
        ban_type: BanType;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="actionType-label">Action Type</InputLabel>
            <Select<BanType>
                fullWidth
                label={'Action Type'}
                labelId="actionType-label"
                id="ban_type"
                name={'ban_type'}
                value={formik.values.ban_type}
                onChange={formik.handleChange}
                error={
                    formik.touched.ban_type && Boolean(formik.errors.ban_type)
                }
                defaultValue={BanType.Banned}
            >
                {[BanType.Banned, BanType.NoComm].map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v == BanType.NoComm ? 'Mute' : 'Ban'}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.ban_type && formik.errors.ban_type}
            </FormHelperText>
        </FormControl>
    );
};
