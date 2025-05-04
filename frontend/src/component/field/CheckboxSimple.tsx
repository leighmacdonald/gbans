import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import { Updater } from '@tanstack/react-form';

type Props = {
    readonly label?: string;
    handleChange?: (updater: Updater<boolean>) => void;
    handleBlur?: () => void;
} & CheckboxProps;

export const CheckboxSimple = ({ label, onChange, value, onBlur }: Props) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox onChange={onChange} onBlur={onBlur} checked={Boolean(value)} />}
                label={label}
            />
        </FormGroup>
    );
};
