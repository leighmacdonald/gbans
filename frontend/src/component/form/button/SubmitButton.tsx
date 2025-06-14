import CheckIcon from '@mui/icons-material/Check';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';

type Props = {
    label?: string;
    labelLoading?: string;
} & ButtonProps;

export const SubmitButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button {...props} type="submit" startIcon={<CheckIcon />} loading={isSubmitting} color={'success'}>
                    {props.label ?? 'Submit'}
                </Button>
            )}
        </form.Subscribe>
    );
};
