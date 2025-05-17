import { ReactNode } from 'react';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';
import { defaultButtonVariant } from '../../../theme.ts';

type Props = {
    label?: string;
    labelLoading?: string;
    disabled?: boolean;
    startIcon?: ReactNode;
    endIcon?: ReactNode;
} & ButtonProps;

export const ResetButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button {...props} type="reset" color={'warning'} variant={defaultButtonVariant}>
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Reset')}
                </Button>
            )}
        </form.Subscribe>
    );
};
