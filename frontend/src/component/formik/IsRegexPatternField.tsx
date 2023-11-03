import React from 'react';
import Checkbox from '@mui/material/Checkbox';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import { useFormikContext } from 'formik';

export interface IsRegexPatternFieldProps {
    is_regex: boolean;
}

export const IsRegexPatternField = <T,>() => {
    const { values, touched, errors, handleChange, isSubmitting } =
        useFormikContext<T & IsRegexPatternFieldProps>();
    return (
        <FormControl fullWidth>
            <FormGroup>
                <FormControlLabel
                    disabled={isSubmitting}
                    control={
                        <Checkbox
                            name={'is_regex'}
                            checked={values.is_regex}
                            onChange={handleChange}
                        />
                    }
                    label={'Regular Expression'}
                />
            </FormGroup>
            <FormHelperText>
                {touched.is_regex && errors.is_regex && `${errors.is_regex}`}
            </FormHelperText>
        </FormControl>
    );
};
