import { ReactNode } from 'react';
import ClearIcon from '@mui/icons-material/Clear';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';

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
            {() => (
                <Button {...props} type="button" color={'secondary'} startIcon={<ClearIcon />}>
                    {props.label ?? 'Clear'}
                </Button>
            )}
        </form.Subscribe>
    );
};
