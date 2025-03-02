import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';

type Props = {
    disabled?: boolean;
    readonly label?: string;
};

export const CheckboxSimple = (props: Props & CheckboxProps) => {
    return (
        <FormGroup>
            <FormControlLabel disabled={props.disabled} control={<Checkbox {...props} />} label={props.label} />
        </FormGroup>
    );
};
