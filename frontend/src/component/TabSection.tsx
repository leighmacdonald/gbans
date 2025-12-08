import { PropsWithChildren } from 'react';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';

export const TabSection = <Tabs,>({
    children,
    tab,
    currentTab,
    label,
    description
}: PropsWithChildren & {
    tab: Tabs;
    currentTab: Tabs;
    label: string;
    description: string;
}) => {
    return (
        <Grid size={{ xs: 8, sm: 9, md: 10 }} display={tab == currentTab ? undefined : 'none'} marginTop={1}>
            <Typography variant={'h1'}>{label}</Typography>
            <Typography variant={'subtitle1'} marginBottom={2}>
                {description}
            </Typography>
            {children}
        </Grid>
    );
};
