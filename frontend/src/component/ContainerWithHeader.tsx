import React from 'react';
import { Heading } from './Heading';
import Stack from '@mui/material/Stack';
import Paper from '@mui/material/Paper';

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
}

export const ContainerWithHeader = ({
    title,
    children,
    iconLeft,
    iconRight,
    align = 'flex-start',
    spacing = 2,
    elevation = 1,
    marginTop = 2
}: ContainerWithHeaderProps) => {
    return (
        <Paper elevation={elevation}>
            <Heading iconLeft={iconLeft} iconRight={iconRight} align={align}>
                {title}
            </Heading>
            <Stack spacing={spacing} sx={{ marginTop }}>
                {children}
            </Stack>
        </Paper>
    );
};
