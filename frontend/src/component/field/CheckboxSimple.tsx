import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import { FieldProps } from './common.ts';

export const CheckboxSimple = ({
    handleBlur,
    handleChange,
    state,
    label,
    onChange,
    disabled = false
}: FieldProps<boolean>) => {
    return (
        <FormGroup>
            <FormControlLabel
                disabled={disabled}
                control={
                    <Checkbox
                        checked={state.value}
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
