import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const NoteFieldValidator = yup.string().label('Hidden Moderator Note');

export interface NoteInputFieldProps {
    note: string;
}

export const NoteField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<T & NoteInputFieldProps>();
    return (
        <TextField
            fullWidth
            id="note"
            name={'note'}
            label="Moderator Notes (hidden from public)"
            multiline
            value={values.note}
            onChange={handleChange}
            error={touched.note && Boolean(errors.note)}
            helperText={touched.note && errors.note && `${errors.note}`}
            rows={10}
            variant="outlined"
        />
    );
};
