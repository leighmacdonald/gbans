import { DesktopDateTimePicker, DesktopDateTimePickerProps } from '@mui/x-date-pickers';
import { useStore } from '@tanstack/react-form';
import { useFieldContext } from '../../../contexts/formContext.tsx';

type Props = {} & DesktopDateTimePickerProps;

export const DateTimeField = (props: Props) => {
    const field = useFieldContext<Date>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    return (
        <DesktopDateTimePicker
            {...props}
            value={field.state.value}
            formatDensity={'spacious'}
            minDate={new Date()}
            slotProps={{
                textField: {
                    variant: 'outlined',
                    error: errors.length > 0,
                    helperText: errors.map(String).join(', ')
                }
            }}
        />
    );
};
