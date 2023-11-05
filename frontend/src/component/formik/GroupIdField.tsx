import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const GroupIdFieldValidator = yup
    .string()
    .length(18, 'Must be positive integer');

interface GroupIDFieldProps {
    group_id: string;
    ban_group_id?: number;
}

export const GroupIdField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & GroupIDFieldProps
    >();
    return (
        <TextField
            fullWidth
            disabled={
                values.ban_group_id != undefined && values.ban_group_id > 0
            }
            id="group_id"
            name={'group_id'}
            label="Steam Group ID"
            value={values.group_id}
            onChange={handleChange}
            error={touched.group_id && Boolean(errors.group_id)}
            helperText={
                touched.group_id && errors.group_id && `${errors.group_id}`
            }
            variant="outlined"
        />
    );
};
