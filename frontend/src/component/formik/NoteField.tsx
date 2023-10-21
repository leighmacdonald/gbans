import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import React from 'react';
import * as yup from 'yup';

export const NoteFieldValidator = yup.string().label('Hidden Moderator Note');

export const NoteField = ({
    formik
}: {
    formik: FormikState<{
        note: string;
    }> &
        FormikHandlers;
}) => {
    return (
        <TextField
            fullWidth
            id="note"
            name={'note'}
            label="Moderator Notes (hidden from public)"
            multiline
            value={formik.values.note}
            onChange={formik.handleChange}
            error={formik.touched.note && Boolean(formik.errors.note)}
            helperText={formik.touched.note && formik.errors.note}
            rows={10}
            variant="outlined"
        />
    );
};
