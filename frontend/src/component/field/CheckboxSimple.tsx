import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';

type Props = {
    readonly label?: string;
} & CheckboxProps;

export const CheckboxSimple = ({ checked, onChange, onBlur, label }: Props) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={<Checkbox onChange={(e, v) => onChange && onChange(e, v)} onBlur={onBlur} checked={checked} />}
                label={label}
            />
        </FormGroup>
    );
};
