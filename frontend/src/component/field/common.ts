import { Updater } from '@tanstack/react-form';

export type FieldProps<T = string> = {
    disabled?: boolean;
    readonly label?: string;

    handleChange: (updater: Updater<T>) => void;
    handleBlur: () => void;
    readonly fullwidth?: boolean;
    onChange?: (value: T) => void;
    multiline?: boolean;
    rows?: number;
    placeholder?: string;
    isValidating?: boolean;
    isTouched?: boolean;
};
