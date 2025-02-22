import { forwardRef, JSX, SyntheticEvent, useState } from 'react';
import { AlertColor } from '@mui/material/Alert';
import MuiAlert, { AlertProps } from '@mui/material/Alert';
import Snackbar, { SnackbarOrigin } from '@mui/material/Snackbar';
import Typography from '@mui/material/Typography';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';

export interface Flash {
    level: AlertColor;
    heading: string;
    message: string;
    closable?: boolean;
    link_to?: string;
}

const Alert = forwardRef<HTMLDivElement, AlertProps>(function Alert(props, ref) {
    return <MuiAlert elevation={6} ref={ref} variant="filled" {...props} />;
});

interface State extends SnackbarOrigin {
    open: boolean;
}

export const PositionedSnackbar = ({ notification }: { notification: Flash }) => {
    const [state, setState] = useState<State>({
        open: true,
        vertical: 'bottom',
        horizontal: 'left'
    });

    const handleClose = (_: SyntheticEvent | Event, reason?: string) => {
        if (reason === 'clickaway') {
            return;
        }

        setState({ ...state, open: false });
    };

    return (
        <Snackbar open={state.open} autoHideDuration={10000} onClose={handleClose}>
            <Alert
                severity={notification.level}
                sx={{ width: '100%' }}
                onClose={() => {
                    setState((prevState) => {
                        return { ...prevState, open: false };
                    });
                }}
            >
                {notification.heading && (
                    <Typography fontWeight={700} sx={{ textTransform: 'capitalize' }}>
                        {notification.heading}
                    </Typography>
                )}
                <Typography>{notification.message}</Typography>
            </Alert>
        </Snackbar>
    );
};

export const Flashes = (): JSX.Element => {
    const { flashes } = useUserFlashCtx();

    return (
        <>
            {flashes.map((flash, index) => {
                return <PositionedSnackbar notification={flash} key={`flash-${index}`} />;
            })}
        </>
    );
};
