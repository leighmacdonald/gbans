import FormHelperText from '@mui/material/FormHelperText';
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
    placeholder = undefined,
    errorText
}: FieldProps & TextFieldProps) => {
    return (
        <>
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
                variant="filled"
                error={error}
                helperText={helperText}
            />
            {error ? (
                <FormHelperText role="alert" error={true}>
                    {errorText}
                </FormHelperText>
            ) : null}
        </>
    );
};
