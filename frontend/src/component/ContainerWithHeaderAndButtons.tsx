import { FC, JSX, ReactNode } from 'react';
import Grid from '@mui/material/Grid2';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import { tf2Fonts } from '../theme';
import { VCenteredElement } from './Heading';

interface ContainerWithHeaderProps {
    title: string;
    children?: JSX.Element[] | JSX.Element | string;
    iconLeft?: ReactNode;
    spacing?: number;
    elevation?: number;
    marginTop?: number;
    padding?: number;
    buttons?: ReactNode[];
}

export const ContainerWithHeaderAndButtons = ({
    title,
    children,
    iconLeft,
    spacing = 2,
    elevation = 1,
    marginTop = 0,
    padding = 1,
    buttons
}: ContainerWithHeaderProps) => {
    return (
        <Paper elevation={elevation}>
            <HeadingWithButtons iconLeft={iconLeft} buttons={buttons}>
                {title}
            </HeadingWithButtons>
            <Stack spacing={spacing} sx={{ marginTop }} padding={padding}>
                {children}
            </Stack>
        </Paper>
    );
};

interface HeadingWithButtonsProps {
    children: JSX.Element[] | JSX.Element | string;
    bgColor?: string;
    iconLeft?: ReactNode;
    buttons?: ReactNode[];
}

export const HeadingWithButtons: FC<HeadingWithButtonsProps> = ({
    children,
    bgColor,
    iconLeft,
    buttons
}: HeadingWithButtonsProps) => {
    const theme = useTheme();
    return (
        <Grid
            container
            direction="row"
            alignItems="center"
            //justifyContent={align ?? 'flex-start'}
            padding={1}
            sx={{
                backgroundColor: bgColor ?? theme.palette.primary.main,
                color: theme.palette.common.white,
                ...tf2Fonts
            }}
        >
            {iconLeft && (
                <Grid size="auto" paddingRight={1}>
                    <VCenteredElement icon={iconLeft} />
                </Grid>
            )}

            <Grid size="auto">{children}</Grid>
            {buttons && <Grid size="auto">{buttons}</Grid>}
        </Grid>
    );
};
