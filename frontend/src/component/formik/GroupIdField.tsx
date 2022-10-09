import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import TextField from '@mui/material/TextField';

export const GroupIdField = ({
    formik
}: {
    formik: FormikState<{
        groupId: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            id="groupId"
            name={'groupId'}
            label="Steam Group ID"
            value={formik.values.groupId}
            onChange={formik.handleChange}
            error={formik.touched.groupId && Boolean(formik.errors.groupId)}
            helperText={formik.touched.groupId && formik.errors.groupId}
            variant="outlined"
        />
    );
};
