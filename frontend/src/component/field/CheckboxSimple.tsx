import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';

type Props = {
    readonly label?: string;
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
