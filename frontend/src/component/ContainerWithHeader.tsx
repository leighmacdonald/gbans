import { PropsWithChildren, ReactNode } from 'react';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Heading } from './Heading';

interface ContainerWithHeaderProps {
    title: string;
    iconLeft?: ReactNode;
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
}: PropsWithChildren<ContainerWithHeaderProps>) => {
    return (
        <Paper elevation={elevation}>
            <Heading iconLeft={iconLeft}>{title}</Heading>
            <Stack spacing={spacing} sx={{ marginTop }} padding={padding}>
                {children}
            </Stack>
        </Paper>
    );
};
