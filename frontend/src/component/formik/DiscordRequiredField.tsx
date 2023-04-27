import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';
import Switch from '@mui/material/Switch';
import FormGroup from '@mui/material/FormGroup';
import FormControlLabel from '@mui/material/FormControlLabel';

export const discordRequiredValidator = yup
    .boolean()
    .label('Is discordutil required')
    .required();

export const DiscordRequiredField = ({
    formik
}: {
    formik: FormikState<{
        discord_required: boolean;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <FormGroup>
                <FormControlLabel
                    disabled={formik.isSubmitting}
                    id={'discord_required'}
                    name={'discord_required'}
                    onChange={formik.handleChange}
                    control={<Switch defaultChecked />}
                    label="Discord Required"
                />
            </FormGroup>
            <FormHelperText>
                {formik.touched.discord_required &&
                    formik.errors.discord_required}
            </FormHelperText>
        </FormControl>
    );
};
