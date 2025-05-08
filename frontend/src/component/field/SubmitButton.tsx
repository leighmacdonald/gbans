import { ReactNode } from 'react';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../contexts/formContext.tsx';

type Props = {
    label?: string;
    labelLoading?: string;
    disabled?: boolean;
    startIcon?: ReactNode;
    endIcon?: ReactNode;
} & ButtonProps;

export const SubmitButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button {...props} type="submit">
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Submit')}
                </Button>
            )}
        </form.Subscribe>
    );
};
