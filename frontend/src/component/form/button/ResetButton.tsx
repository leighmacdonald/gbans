import { ReactNode } from 'react';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';
import { variant } from './index.ts';

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
                <Button
                    {...props}
                    type="reset"
                    variant={variant}
                    color={'warning'}
                    startIcon={props.startIcon ?? <RestartAltIcon />}
                >
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Reset')}
                </Button>
            )}
        </form.Subscribe>
    );
};
