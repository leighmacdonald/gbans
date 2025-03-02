import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';

type Props = {
    readonly label?: string;
    handleChange: (v: boolean) => void;
    handleBlur: () => void;
} & CheckboxProps;

export const CheckboxSimple = (props: Props) => {
    return (
        <FormGroup>
            <FormControlLabel
                control={
                    <Checkbox
                        {...props}
                        onChange={(e) => props.handleChange(e.target.checked)}
                        onBlur={props.handleBlur}
                    />
                }
                label={props.label}
            />
        </FormGroup>
    );
};
