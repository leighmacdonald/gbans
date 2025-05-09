import FormHelperText from '@mui/material/FormHelperText';
import * as MUITextField from '@mui/material/TextField';
import { TextFieldProps } from '@mui/material/TextField';
import { useStore } from '@tanstack/react-form';
import { useFieldContext } from '../../../contexts/formContext.tsx';

type Props = {
    label: string;
} & TextFieldProps;

export const TextField = (props: Props) => {
    const field = useFieldContext<string>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    return (
        <>
            <MUITextField.default {...props} onChange={(e) => field.handleChange(e.target.value)} variant="filled" />
            {errors ? (
                <FormHelperText role="alert" error={true}>
                    {props.helperText}
                </FormHelperText>
            ) : null}
        </>
    );
};
