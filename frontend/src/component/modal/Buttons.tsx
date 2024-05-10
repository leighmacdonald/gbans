import { ReactNode } from 'react';
import { useModal } from '@ebay/nice-modal-react';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SaveIcon from '@mui/icons-material/Save';
import Button from '@mui/material/Button';

interface onClickProps {
    onClick?: () => void;
    formId?: string;
}

export const CancelButton = ({ onClick }: onClickProps) => {
    const modal = useModal();
    return (
        <Button startIcon={<CloseIcon />} color={'error'} variant={'contained'} onClick={onClick ?? modal.hide}>
            Cancel
        </Button>
    );
};

export const SubmitButton = ({
    formId,
    disabled = false,
    label = 'Save',
    startIcon = <SaveIcon />,
    fullWidth = false
}: onClickProps & {
    disabled?: boolean;
    label?: string;
    startIcon?: ReactNode;
    fullWidth?: boolean;
}) => {
    return (
        <Button
            fullWidth={fullWidth}
            startIcon={startIcon}
            color={'success'}
            variant={'contained'}
            disabled={disabled}
            type={'submit'}
            form={formId}
            size={'large'}
            sx={{ height: '52px' }}
        >
            {label}
        </Button>
    );
};
export const ConfirmButton = ({ onClick, disabled = false }: onClickProps & { disabled?: boolean }) => {
    return (
        <Button
            startIcon={<CheckIcon />}
            color={'success'}
            variant={'contained'}
            onClick={onClick ?? undefined}
            disabled={disabled}
        >
            Confirm
        </Button>
    );
};

export const CloseButton = ({ onClick }: onClickProps) => {
    return (
        <Button startIcon={<ClearIcon />} color={'warning'} variant={'contained'} onClick={onClick}>
            Close
        </Button>
    );
};

export const ResetButton = ({ formId, disabled = false }: onClickProps & { disabled?: boolean }) => {
    return (
        <Button
            startIcon={<RestartAltIcon />}
            color={'warning'}
            variant={'contained'}
            type={'reset'}
            form={formId}
            disabled={disabled}
        >
            Reset
        </Button>
    );
};
