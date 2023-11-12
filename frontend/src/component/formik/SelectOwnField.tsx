import React from 'react';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { VCenterBox } from '../VCenterBox';

export const selectOwnValidator = yup
    .boolean()
    .label('Include only results with yourself')
    .required();

interface SelectOwnFieldProps {
    select_own: boolean;
    source_id?: string;
}

export const SelectOwnField = <T,>({ disabled }: { disabled?: boolean }) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & SelectOwnFieldProps
    >();
    return (
        <VCenterBox>
            <FormControl fullWidth disabled={disabled}>
                <FormGroup>
                    <FormControlLabel
                        value={values.select_own}
                        id={'select_own'}
                        name={'select_own'}
                        onChange={handleChange}
                        control={<Switch checked={values.select_own} />}
                        label="Only Mine"
                    />
                </FormGroup>
                <FormHelperText>
                    {touched.select_own &&
                        errors.select_own &&
                        `${errors.select_own}`}
                </FormHelperText>
            </FormControl>
        </VCenterBox>
    );
};
