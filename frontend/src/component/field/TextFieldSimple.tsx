import TextField, { TextFieldProps } from '@mui/material/TextField';
import { FieldProps } from './common.ts';

export const TextFieldSimple = ({
    label,
    handleChange,
    handleBlur,
    fullwidth = true,
    disabled = false,
    rows = 1,
    value,
    error,
    helperText,
    placeholder = undefined
}: FieldProps & TextFieldProps) => {
    return (
        <TextField
            multiline={rows > 1}
            rows={rows > 1 ? rows : undefined}
            disabled={disabled}
            fullWidth={fullwidth}
            label={label}
            value={value}
            placeholder={placeholder}
            onChange={(e) => handleChange(e.target.value)}
            onBlur={handleBlur}
            variant="outlined"
            error={error}
            helperText={helperText}
        />
    );
};
