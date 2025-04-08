import { TextFieldProps } from '@mui/material/TextField';
import { DesktopDateTimePicker } from '@mui/x-date-pickers';
import { parseISO } from 'date-fns';
import { FieldProps } from './common.ts';

export const DateTimeSimple = ({
    label,
    value,
    disabled,
    handleChange,
    error,
    helperText
}: FieldProps & TextFieldProps) => {
    return (
        <DesktopDateTimePicker
            disabled={disabled}
            label={label}
            value={parseISO(value as string)}
            formatDensity={'spacious'}
            minDate={new Date()}
            onChange={(e) => handleChange(e ? e.toISOString() : '')}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: error,
                    helperText: helperText
                }
            }}
        />
    );
};
