import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import React, { JSX } from 'react';
import { Heading } from './Heading';

export type JustifyTypes =
    | 'flex-start'
    | 'center'
    | 'flex-end'
    | 'space-between';

interface ContainerWithHeaderProps {
    title: string;
    children?: JSX.Element[] | JSX.Element | string;
    iconLeft?: React.ReactNode;
    iconRight?: React.ReactNode;
    align?: JustifyTypes;
    spacing?: number;
    elevation?: number;
    marginTop?: number;
    padding?: number;
}

export const ContainerWithHeader = ({
    title,
    children,
    iconLeft,
    iconRight,
    align = 'flex-start',
    spacing = 2,
    elevation = 1,
    marginTop = 0,
    padding = 1
}: ContainerWithHeaderProps) => {
    return (
        <Paper elevation={elevation}>
            <Heading iconLeft={iconLeft} iconRight={iconRight} align={align}>
                {title}
            </Heading>
            <Stack spacing={spacing} sx={{ marginTop }} padding={padding}>
                {children}
            </Stack>
        </Paper>
    );
};
