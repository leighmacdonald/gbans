import { DesktopDateTimePicker } from '@mui/x-date-pickers';
import { parseISO } from 'date-fns';
import { FieldProps } from './common.ts';

export const DateTimeSimple = ({ label, state, disabled, handleChange }: FieldProps) => {
    return (
        <DesktopDateTimePicker
            disabled={disabled}
            label={label}
            value={parseISO(state.value)}
            formatDensity={'spacious'}
            minDate={new Date()}
            onChange={(e) => handleChange(e ? e.toISOString() : '')}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: state.meta.touchedErrors.length > 0,
                    helperText: state.meta.errors.join(',')
                }
            }}
        />
    );
};
