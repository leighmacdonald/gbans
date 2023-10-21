import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export const discordRequiredValidator = yup
    .boolean()
    .label('Is discord required')
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
