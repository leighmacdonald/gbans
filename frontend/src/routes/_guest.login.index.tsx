import DoDisturbIcon from '@mui/icons-material/DoDisturb';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { createFileRoute, useRouteContext } from '@tanstack/react-router';
import { z } from 'zod/v4';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import steamLogo from '../icons/steam_login_lg.png';
import { generateOIDCLink } from '../util/auth/generateOIDCLink.ts';

const loginSearchSchema = z.object({
    redirect: z.string().catch('/')
});

export const Route = createFileRoute('/_guest/login/')({
    component: LoginPage,
    validateSearch: (search) => loginSearchSchema.parse(search),
    beforeLoad: ({ context }) => {
        // Otherwise, return the user in context
        return context.auth;
    }
});

export function LoginPage() {
    const message = 'To access this page, please login using your steam account below.';
    const title = 'Permission Denied';
    const { isAuthenticated } = useRouteContext({ from: '/_guest/login/' });
    const { redirect } = Route.useSearch();

    return (
        <Grid container justifyContent={'center'} alignItems={'center'}>
            <Grid size={{ xs: 12 }}>
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
                                <Stack
                                    justifyContent="center"
                                    gap={2}
                                    flexDirection="row"
                                    width={1.0}
                                    flexWrap="wrap"
                                    padding={2}
                                >
                                    <Button
                                        sx={{ alignSelf: 'center' }}
                                        component={Link}
                                        href={generateOIDCLink(redirect)}
                                    >
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
