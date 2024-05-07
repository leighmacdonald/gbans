import ClearIcon from '@mui/icons-material/Clear';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { useNavigate } from '@tanstack/react-router';

type ButtonProps = {
    canSubmit: boolean;
    isSubmitting: boolean;
    reset: () => void;
    submitLabel?: string;
    resetLabel?: string;
    clearLabel?: string;
    showClear?: boolean;
    closeLabel?: string;
    onClear?: () => Promise<void>;
    onClose?: () => Promise<void>;
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
    onClose
}: ButtonProps) => {
    const navigate = useNavigate();

    return (
        <ButtonGroup>
            <Button
                type="submit"
                disabled={!canSubmit}
                variant={'contained'}
                color={'success'}
                startIcon={<SendIcon />}
            >
                {isSubmitting ? '...' : submitLabel}
            </Button>
            <Button
                type="reset"
                onClick={() => reset()}
                variant={'contained'}
                color={'warning'}
                startIcon={<RestartAltIcon />}
            >
                {resetLabel}
            </Button>
            {showClear ||
                (onClear && (
                    <Button
                        type="button"
                        onClick={async () => {
                            if (onClear) {
                                return await onClear();
                            }
                            await navigate({
                                search: (prev) => {
                                    return {
                                        ...prev,
                                        page: 0,
                                        steam_id: '',
                                        body: '',
                                        persona_name: '',
                                        server_id: 0
                                    };
                                }
                            });
                            // TODO fix this hackjob
                            window.location.reload();
                        }}
                        variant={'contained'}
                        color={'error'}
                        startIcon={<ClearIcon />}
                    >
                        {clearLabel}
                    </Button>
                ))}
            {onClose && (
                <Button onClick={onClose} variant={'contained'} color={'error'} startIcon={<CloseIcon />}>
                    {closeLabel}
                </Button>
            )}
        </ButtonGroup>
    );
};
