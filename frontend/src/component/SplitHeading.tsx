import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';
import { FC } from 'react';
import React from 'react';
import Stack from '@mui/material/Stack';

interface SplitHeadingProps {
    left: string;
    right: string;
    bgColor?: string;
}

export const SplitHeading: FC<SplitHeadingProps> = ({
    left,
    right,
    bgColor
}: SplitHeadingProps) => {
    const theme = useTheme();
    return (
        <Stack direction={'row'}>
            <Typography
                variant={'h6'}
                align={'left'}
                paddingTop={1}
                paddingBottom={1}
                paddingLeft={2}
                sx={{
                    backgroundColor: bgColor ?? theme.palette.primary.main,
                    color: theme.palette.background.paper,
                    width: '100%'
                }}
            >
                {left}
            </Typography>
            <Typography
                variant={'h6'}
                align={'right'}
                paddingTop={1}
                paddingBottom={1}
                paddingRight={2}
                sx={{
                    backgroundColor: bgColor ?? theme.palette.primary.main,
                    color: theme.palette.background.paper,
                    width: 200
                }}
            >
                {right}
            </Typography>
        </Stack>
    );
};
