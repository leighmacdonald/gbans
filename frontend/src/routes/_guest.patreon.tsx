import PaymentIcon from '@mui/icons-material/Payment';
import SearchIcon from '@mui/icons-material/Search';
import SettingsInputComponentIcon from '@mui/icons-material/SettingsInputComponent';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { createFileRoute, Navigate } from '@tanstack/react-router';
import Image from 'mui-image';
import { z } from 'zod';
import { apiGetPatreonLogin, apiGetPatreonCampaigns } from '../api/patreon.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { MarkDownRenderer } from '../component/MarkdownRenderer.tsx';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';
import { ensureFeatureEnabled } from '../util/features.ts';

const patreonSearchSchema = z.object({
    redirect: z.string().catch('/')
});

export const Route = createFileRoute('/_guest/patreon')({
    component: Patreon,
    beforeLoad: () => {
        ensureFeatureEnabled('patreon_enabled');
    },
    validateSearch: (search) => patreonSearchSchema.parse(search),
    loader: async ({ context }) => {
        const campaign = await context.queryClient.fetchQuery({
            queryKey: ['patreonCampaign'],
            queryFn: apiGetPatreonCampaigns
        });

        return { campaign };
    }
});

function Patreon() {
    const { isAuthenticated, queryClient, profile } = Route.useRouteContext();
    const { campaign } = Route.useLoaderData();
    const theme = useTheme();
    const { appInfo } = useAppInfoCtx();
    const followCallback = async () => {
        const result = await queryClient.fetchQuery({ queryKey: ['callback'], queryFn: apiGetPatreonLogin });
        window.open(result.url, '_self');
    };

    if (!appInfo.patreon_enabled) {
        return <Navigate to={'/'} />;
    }

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={`Patreon Campaign: ${campaign.attributes.creation_name}`}
                    iconLeft={<PaymentIcon />}
                    buttons={
                        profile.patreon_id
                            ? []
                            : [
                                  <Button
                                      key={'connect'}
                                      variant={'contained'}
                                      color={'success'}
                                      disabled={!isAuthenticated() || profile.patreon_id != ''}
                                      onClick={followCallback}
                                      startIcon={<SettingsInputComponentIcon />}
                                  >
                                      Connect Patreon
                                  </Button>
                              ]
                    }
                >
                    <Grid container>
                        <Grid xs={12}>
                            <Stack spacing={1}>
                                <Paper>
                                    <Image
                                        height={'100%'}
                                        width={'100%'}
                                        alt={'Campaign background'}
                                        src={campaign.attributes.image_url}
                                    />
                                </Paper>

                                <MarkDownRenderer body_md={campaign.attributes.summary} />

                                <MarkDownRenderer body_md={campaign.attributes.thanks_msg} />
                            </Stack>
                        </Grid>
                        <Grid xs={12}>
                            <Box display="flex" justifyContent="center" alignItems="center" padding={2}>
                                <Paper
                                    elevation={1}
                                    sx={{
                                        backgroundColor: theme.palette.primary.main,
                                        color: theme.palette.common.white,
                                        borderRadius: 0.5
                                    }}
                                >
                                    <Typography
                                        variant={'subtitle1'}
                                        textAlign={'center'}
                                        padding={2}
                                        textTransform={'uppercase'}
                                    >
                                        Patrons
                                    </Typography>
                                    <Typography
                                        variant={'h1'}
                                        textAlign={'center'}
                                        padding={2}
                                        sx={{ backgroundColor: theme.palette.primary.light }}
                                    >
                                        {campaign.attributes.patron_count}
                                    </Typography>
                                </Paper>
                            </Box>
                        </Grid>
                        <Grid xs={12}>
                            <Box textAlign={'center'}>
                                <Button
                                    component={Link}
                                    variant={'contained'}
                                    color={'success'}
                                    startIcon={<SearchIcon />}
                                    href={campaign.attributes.url + '/membership'}
                                >
                                    View Membership Tiers
                                </Button>
                            </Box>
                        </Grid>
                    </Grid>
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}
