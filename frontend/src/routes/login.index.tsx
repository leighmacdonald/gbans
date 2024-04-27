import DoDisturbIcon from '@mui/icons-material/DoDisturb';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute, redirect, useRouteContext } from '@tanstack/react-router';
import { generateOIDCLink } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import steamLogo from '../icons/steam_login_lg.png';

export const Route = createFileRoute('/login/')({
    component: LoginPage,
    beforeLoad: ({ context }) => {
        if (context.auth.isAuthenticated()) {
            throw redirect({
                to: '/',
                search: {
                    // Use the current location to power a redirect after login
                    // (Do not use `router.state.resolvedLocation` as it can
                    // potentially lag behind the actual current location)
                    redirect: location.href
                }
            });
        }
        // Otherwise, return the user in context
        return context.auth;
    }
});

export interface LoginFormProps {
    message?: string;
    title?: string;
}

export function LoginPage() {
    const message = 'To access this page, please login using your steam account below.';
    const title = 'Permission Denied';
    const { isAuthenticated } = useRouteContext({ from: '/login/' });

    return (
        <Grid container justifyContent={'center'} alignItems={'center'}>
            <Grid xs={12}>
                <ContainerWithHeader title={title} iconLeft={<DoDisturbIcon />}>
                    <>
                        {isAuthenticated() && (
                            <Typography variant={'body1'} padding={2}>
                                Insufficient permission to access this page.
                            </Typography>
                        )}
                        {!isAuthenticated() && (
                            <>
                                <Typography variant={'body1'} padding={2} paddingBottom={0}>
                                    {message}
                                </Typography>
                                <Stack justifyContent="center" gap={2} flexDirection="row" width={1.0} flexWrap="wrap" padding={2}>
                                    <Button sx={{ alignSelf: 'center' }} component={Link} href={generateOIDCLink(window.location.pathname)}>
                                        <img src={steamLogo} alt={'Steam Login'} />
                                    </Button>
                                </Stack>
                            </>
                        )}
                    </>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}
