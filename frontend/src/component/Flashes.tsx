import React from 'react';
import Alert from '@mui/lab/Alert';
import { AlertColor } from '@mui/material/Alert/Alert';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export interface Flash {
    level: AlertColor;
    heading: string;
    message: string;
    closable?: boolean;
    link_to?: string;
}

export interface FlashesProps {
    flashes: Flash[];
}

export const Flashes = (): JSX.Element => {
    const { flashes, setFlashes } = useUserFlashCtx();
    return (
        <>
            {flashes.map((f, i) => {
                return (
                    <Alert
                        key={`alert-${i}`}
                        severity={f.level}
                        onClose={() => {
                            setFlashes(
                                flashes.filter(
                                    (flash) => flash.message != f.message
                                )
                            );
                        }}
                    >
                        {f.message}
                    </Alert>
                );
            })}
        </>
    );
};
