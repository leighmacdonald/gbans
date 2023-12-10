import React, { JSX } from 'react';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Heading } from './Heading';

export type JustifyTypes =
    | 'flex-start'
    | 'center'
    | 'flex-end'
    | 'space-between';

interface ContainerWithHeaderProps {
    title: string;
    children?: JSX.Element[] | JSX.Element | string | boolean;
    iconLeft?: React.ReactNode;
    spacing?: number;
    elevation?: number;
    marginTop?: number;
    padding?: number;
}

export const ContainerWithHeader = ({
    title,
    children,
    iconLeft,
    spacing = 2,
    elevation = 1,
    marginTop = 0,
    padding = 1
}: ContainerWithHeaderProps) => {
    return (
        <Paper elevation={elevation}>
            <Heading iconLeft={iconLeft}>{title}</Heading>
            <Stack spacing={spacing} sx={{ marginTop }} padding={padding}>
                {children}
            </Stack>
        </Paper>
    );
};
