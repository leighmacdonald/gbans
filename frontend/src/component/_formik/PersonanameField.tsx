import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export interface PersonanameFieldProps {
    personaname: string;
}

export const PersonanameField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<T & PersonanameFieldProps>();
    return (
        <TextField
            fullWidth
            id="personaname"
            name={'personaname'}
            label="Name"
            value={values.personaname}
            onChange={handleChange}
            error={touched.personaname && Boolean(errors.personaname)}
            helperText={touched.personaname && errors.personaname && `${errors.personaname}`}
            variant="outlined"
        />
    );
};
