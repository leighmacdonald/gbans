import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import { BanType } from '../../api';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';

export const BanTypeField = ({
    formik
}: {
    formik: FormikState<{
        banType: BanType;
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
                id="banType"
                name={'banType'}
                value={formik.values.banType}
                onChange={formik.handleChange}
                error={formik.touched.banType && Boolean(formik.errors.banType)}
                defaultValue={BanType.Banned}
            >
                {[BanType.Banned, BanType.NoComm].map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v == BanType.NoComm ? 'Mute' : 'Ban'}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.banType && formik.errors.banType}
            </FormHelperText>
        </FormControl>
    );
};
