import { DateTimePicker } from '@mui/x-date-pickers';
import { parseISO } from 'date-fns';
import { FieldProps } from './common.ts';

export const DateTimeSimple = ({ label, state, disabled, handleChange }: FieldProps) => {
    return (
        <DateTimePicker
            disabled={disabled}
            label={label}
            value={parseISO(state.value)}
            formatDensity={'spacious'}
            //onError={(newError) => setError(newError)}
            onChange={(e) => handleChange(e ? e.toString() : '')}
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
