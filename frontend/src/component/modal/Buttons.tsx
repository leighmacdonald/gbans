import React from 'react';
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
    return (
        <Button
            startIcon={<CloseIcon />}
            color={'error'}
            variant={'contained'}
            onClick={onClick}
        >
            Cancel
        </Button>
    );
};

export const SaveButton = ({
    onClick,
    formId,
    disabled = false
}: onClickProps & { disabled?: boolean }) => {
    return (
        <Button
            startIcon={<SaveIcon />}
            color={'success'}
            variant={'contained'}
            onClick={onClick ?? undefined}
            disabled={disabled}
            type={'submit'}
            form={formId}
        >
            Save
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

export const ClearButton = ({ onClick }: onClickProps) => {
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
    onClick,
    formId,
    disabled = false
}: onClickProps & { disabled?: boolean }) => (
    <Button
        onClick={onClick}
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
