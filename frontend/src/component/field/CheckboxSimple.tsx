import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import { Updater } from '@tanstack/react-form';

type Props<T = boolean> = {
    disabled?: boolean;
    readonly label?: string;
    checked?: boolean;
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

export const CheckboxSimple = ({ handleBlur, handleChange, checked, label, onChange, disabled = false }: Props) => {
    return (
        <FormGroup>
            <FormControlLabel
                disabled={disabled}
                control={
                    <Checkbox
                        checked={checked}
                        onBlur={handleBlur}
                        onChange={(_, v) => {
                            handleChange(v);
                            if (onChange) {
                                onChange(v);
                            }
                        }}
                    />
                }
                label={label}
            />
        </FormGroup>
    );
};
