import FormHelperText from '@mui/material/FormHelperText';
import * as MUITextField from '@mui/material/TextField';
import { TextFieldProps } from '@mui/material/TextField';
import { useFieldContext } from '../../contexts/formContext.tsx';

type Props = {
    label: string;
} & TextFieldProps;

export const TextField = (props: Props) => {
    const field = useFieldContext<string>();

    return (
        <>
            <MUITextField.default {...props} onChange={(e) => field.handleChange(e.target.value)} variant="filled" />
            {props.error ? (
                <FormHelperText role="alert" error={true}>
                    {props.helperText}
                </FormHelperText>
            ) : null}
        </>
    );
};
