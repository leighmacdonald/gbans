import { ReactNode } from 'react';
import ClearIcon from '@mui/icons-material/Clear';
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

export const ClearButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button
                    {...props}
                    type="button"
                    variant={variant}
                    color={'secondary'}
                    startIcon={props.startIcon ?? <ClearIcon />}
                >
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Clear')}
                </Button>
            )}
        </form.Subscribe>
    );
};
