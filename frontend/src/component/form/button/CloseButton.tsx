import { ReactNode, useCallback } from 'react';
import { useModal } from '@ebay/nice-modal-react';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';

type Props = {
    label?: string;
    labelLoading?: string;
    disabled?: boolean;
    startIcon?: ReactNode;
    endIcon?: ReactNode;
    onClick?: () => void | Promise<void>;
} & ButtonProps;

export const CloseButton = (props: Props) => {
    const form = useFormContext();
    const modal = useModal();

    const onClick = useCallback(async () => {
        if (modal) {
            await modal.hide();
        }
    }, [modal]);

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button {...props} onClick={props.onClick ?? onClick} type="button">
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Close')}
                </Button>
            )}
        </form.Subscribe>
    );
};
