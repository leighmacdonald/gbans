import CheckIcon from '@mui/icons-material/Check';
import Button, { ButtonProps } from '@mui/material/Button';
import { useFormContext } from '../../../contexts/formContext.tsx';
import { variant } from './index.ts';

type Props = {
    label?: string;
    labelLoading?: string;
} & ButtonProps;

export const SubmitButton = (props: Props) => {
    const form = useFormContext();

    return (
        <form.Subscribe selector={(state) => state.isSubmitting}>
            {(isSubmitting) => (
                <Button
                    {...props}
                    type="submit"
                    variant={variant}
                    color={'success'}
                    startIcon={props.startIcon ?? <CheckIcon />}
                >
                    {isSubmitting ? (props.labelLoading ?? '...') : (props.label ?? 'Submit')}
                </Button>
            )}
        </form.Subscribe>
    );
};
