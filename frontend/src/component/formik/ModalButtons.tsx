import { useModal } from '@ebay/nice-modal-react';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import { LoadingButton } from '@mui/lab';
import { DialogActions } from '@mui/material';
import Button from '@mui/material/Button';

interface ModalButtonsProps {
    modalID?: string;

    inProgress?: boolean;
}

export const ModalButtons = ({ modalID, inProgress }: ModalButtonsProps) => {
    const modal = useModal();
    return (
        <DialogActions>
            {inProgress != undefined && (
                <LoadingButton
                    loading={inProgress}
                    variant={'contained'}
                    color={'success'}
                    startIcon={<CheckIcon />}
                    type={'submit'}
                    form={modalID}
                >
                    Accept
                </LoadingButton>
            )}
            <Button
                variant={'contained'}
                color={'warning'}
                startIcon={<ClearIcon />}
                type={'reset'}
                form={modalID}
            >
                Reset
            </Button>
            <Button
                variant={'contained'}
                color={'error'}
                startIcon={<ClearIcon />}
                type={'button'}
                form={modalID}
                onClick={modal.hide}
            >
                Cancel
            </Button>
        </DialogActions>
    );
};
