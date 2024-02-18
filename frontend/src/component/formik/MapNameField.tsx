import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

interface MapNameFieldProps {
    map_name: string;
}

export const MapNameField = () => {
    const { values, handleChange, touched, errors } =
        useFormikContext<MapNameFieldProps>();
    return (
        <FormControl fullWidth>
            <TextField
                fullWidth
                name={'map_name'}
                id={'map_name'}
                label={'Map Name'}
                value={values.map_name}
                onChange={handleChange}
                error={touched.map_name && Boolean(errors.map_name)}
                helperText={touched.map_name && errors.map_name}
            />
        </FormControl>
    );
};
