import React from 'react';
import { useModal } from '@ebay/nice-modal-react';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SaveIcon from '@mui/icons-material/Save';
import Button from '@mui/material/Button';
import { useFormikContext } from 'formik';

interface onClickProps {
    onClick?: () => void;
    formId?: string;
}

export const CancelButton = ({ onClick }: onClickProps) => {
    const modal = useModal();
    return (
        <Button
            startIcon={<CloseIcon />}
            color={'error'}
            variant={'contained'}
            onClick={onClick ?? modal.hide}
        >
            Cancel
        </Button>
    );
};

export const SubmitButton = ({
    onClick,
    formId,
    disabled = false,
    label = 'Save',
    startIcon = <SaveIcon />
}: onClickProps & {
    disabled?: boolean;
    label?: string;
    startIcon?: React.ReactNode;
}) => {
    const { submitForm } = useFormikContext();
    return (
        <Button
            size={'small'}
            startIcon={startIcon}
            color={'success'}
            variant={'contained'}
            onClick={onClick ?? submitForm}
            disabled={disabled}
            type={'submit'}
            form={formId}
        >
            {label}
        </Button>
    );
};
export const ConfirmButton = ({
    onClick,
    disabled = false
}: onClickProps & { disabled?: boolean }) => {
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
        <Button
            startIcon={<ClearIcon />}
            color={'warning'}
            variant={'contained'}
            onClick={onClick}
        >
            Close
        </Button>
    );
};

export const ResetButton = ({
    formId,
    disabled = false
}: onClickProps & { disabled?: boolean }) => {
    const { resetForm } = useFormikContext();

    return (
        <Button
            size={'small'}
            onClick={() => resetForm()}
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
