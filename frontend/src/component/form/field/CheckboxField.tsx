import { ReactNode } from 'react';
import { FormHelperText } from '@mui/material';
import Checkbox, { CheckboxProps } from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { renderHelpText } from './renderHelpText.ts';

type Props = {
    readonly label?: string;
    helperText?: ReactNode;
} & CheckboxProps;

export const CheckboxField = (props: Props) => {
    const field = useFieldContext<boolean>();

    return (
        <FormGroup>
            <FormControlLabel control={<Checkbox {...props} name={field.name} />} label={props.label} />
            <FormHelperText>{renderHelpText([], props.helperText)}</FormHelperText>
        </FormGroup>
    );
};
