import { PropsWithChildren, ReactNode } from 'react';
import Box from '@mui/material/Box';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { tf2Fonts } from '../theme';

interface HeadingProps {
    bgColor?: string;
    iconLeft?: ReactNode;
    iconRight?: ReactNode;
    align?: 'flex-start' | 'center' | 'flex-end' | 'space-between';
}

interface VCenteredImageProps {
    icon: ReactNode;
}

export const VCenteredElement = ({ icon }: VCenteredImageProps) => {
    return (
        <Box display="flex" justifyContent="right" alignItems="center">
            {icon}
        </Box>
    );
};

export const Heading = ({ children, bgColor, iconLeft, iconRight, align }: PropsWithChildren<HeadingProps>) => {
    const theme = useTheme();
    return (
        <Grid
            container
            direction="row"
            alignItems="center"
            justifyContent={align ?? 'flex-start'}
            padding={1}
            sx={{
                backgroundColor: bgColor ?? theme.palette.primary.main,
                color: theme.palette.common.white,
                ...tf2Fonts
            }}
        >
            {iconLeft && (
                <Grid paddingRight={1}>
                    <VCenteredElement icon={iconLeft} />
                </Grid>
            )}
            <Grid>{children}</Grid>
            {iconRight && (
                <Grid paddingLeft={1}>
                    <VCenteredElement icon={iconRight} />
                </Grid>
            )}
        </Grid>
    );
};
