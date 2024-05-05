import TextField from '@mui/material/TextField';
import { FieldProps } from './common.ts';

export const TextFieldSimple = ({ label, state, handleChange, handleBlur, fullwidth = true, disabled = false }: FieldProps) => {
    return (
        <TextField
            disabled={disabled}
            fullWidth={fullwidth}
            label={label}
            value={state.value}
            onChange={(e) => handleChange(e.target.value)}
            onBlur={handleBlur}
            variant="outlined"
            error={state.meta.touchedErrors.length > 0}
            helperText={state.meta.touchedErrors}
        />
    );
};
