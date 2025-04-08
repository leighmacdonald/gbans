import ClearIcon from '@mui/icons-material/Clear';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { ToOptions } from '@tanstack/react-router';
import { ButtonLink } from '../ButtonLink.tsx';

type ButtonProps = {
    canSubmit: boolean;
    isSubmitting: boolean;
    reset: () => void;
    submitLabel?: string;
    resetLabel?: string;
    clearLabel?: string;
    showClear?: boolean;
    showReset?: boolean;
    closeLabel?: string;
    onClear?: () => Promise<void>;
    onClose?: () => Promise<void>;
    fullWidth?: boolean;
    navigateOpts?: ToOptions;
};

export const Buttons = ({
    canSubmit,
    isSubmitting,
    reset,
    onClear,
    submitLabel = 'Submit',
    resetLabel = 'Reset',
    clearLabel = 'Clear',
    closeLabel = 'Close',
    showClear = false,
    showReset = true,
    fullWidth = false,
    onClose,
    navigateOpts
}: ButtonProps) => {
    return (
        <ButtonGroup fullWidth={fullWidth}>
            <Button
                key={'submit-button'}
                type="submit"
                disabled={!canSubmit}
                variant={'contained'}
                color={'success'}
                startIcon={<SendIcon />}
            >
                {isSubmitting ? '...' : submitLabel}
            </Button>
            {showReset && (
                <Button
                    key={'reset-button'}
                    type="reset"
                    onClick={() => reset()}
                    variant={'contained'}
                    color={'warning'}
                    startIcon={<RestartAltIcon />}
                >
                    {resetLabel}
                </Button>
            )}
            {showClear ||
                (onClear && (
                    <ButtonLink
                        {...navigateOpts}
                        key={'clear-button'}
                        type="button"
                        variant={'contained'}
                        color={'error'}
                        startIcon={<ClearIcon />}
                    >
                        {clearLabel}
                    </ButtonLink>
                ))}
            {onClose && (
                <Button
                    key={'close-button'}
                    onClick={onClose}
                    variant={'contained'}
                    color={'error'}
                    startIcon={<CloseIcon />}
                >
                    {closeLabel}
                </Button>
            )}
        </ButtonGroup>
    );
};
