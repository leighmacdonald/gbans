import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { tf2Fonts } from '../theme';

interface SplitHeadingProps {
    left: string;
    right: string;
    bgColor?: string;
}

export const SplitHeading: FC<SplitHeadingProps> = ({ left, right, bgColor }: SplitHeadingProps) => {
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
                    color: theme.palette.common.white,
                    width: '100%',
                    ...tf2Fonts
                }}
            >
                {left}
            </Typography>
            <Typography
                variant={'subtitle1'}
                align={'right'}
                paddingTop={1}
                paddingBottom={1}
                paddingRight={2}
                sx={{
                    backgroundColor: bgColor ?? theme.palette.primary.main,
                    color: theme.palette.common.white,
                    width: 200
                }}
            >
                {right}
            </Typography>
        </Stack>
    );
};
