import React from 'react';
import { Alert } from '@mui/lab';
import { AlertColor } from '@mui/material/Alert/Alert';

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

export const Flashes = ({ flashes }: FlashesProps): JSX.Element => (
    <>
        {flashes.map((f, i) => {
            return (
                <Alert key={`alert-${i}`} color={f.level}>
                    {f.message}
                </Alert>
            );
        })}
    </>
);
