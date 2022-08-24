import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';
import { FC } from 'react';
import React from 'react';

interface HeadingProps {
    children: JSX.Element | string;
    bgColor?: string;
}

export const Heading: FC<HeadingProps> = ({
    children,
    bgColor
}: HeadingProps) => {
    const theme = useTheme();
    return (
        <Typography
            variant={'h6'}
            align={'center'}
            padding={1}
            sx={{
                backgroundColor: bgColor ?? theme.palette.primary.main,
                color: theme.palette.common.white
            }}
        >
            {children}
        </Typography>
    );
};
