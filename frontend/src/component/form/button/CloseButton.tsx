import CloseIcon from '@mui/icons-material/Close';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';
import { variant } from './index.ts';

type Props = {
    label?: string;
    labelLoading?: string;
} & ButtonProps;

export const CloseButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button {...props} type="button" variant={variant} startIcon={props.startIcon ?? <CloseIcon />}>
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Close')}
                </Button>
            )}
        </form.Subscribe>
    );
};
