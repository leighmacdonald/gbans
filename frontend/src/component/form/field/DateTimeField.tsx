import { DesktopDateTimePicker, DesktopDateTimePickerProps } from '@mui/x-date-pickers';
import { useStore } from '@tanstack/react-form';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { defaultFieldVariant } from '../../../theme.ts';
import { renderHelpText } from './renderHelpText.ts';

type Props = { helpText?: string } & DesktopDateTimePickerProps;

export const DateTimeField = (props: Props) => {
    const field = useFieldContext<Date>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    return (
        <DesktopDateTimePicker
            {...props}
            value={field.state.value}
            formatDensity={'spacious'}
            minDate={props.minDate ?? new Date()}
            slotProps={{
                textField: {
                    variant: defaultFieldVariant,
                    error: errors.length > 0,
                    helperText: renderHelpText(errors, props.helpText)
                }
            }}
        />
    );
};
