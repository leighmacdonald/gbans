import { JSX, useMemo } from 'react';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';
import RouterLink from './RouterLink.tsx';

export const Footer = (): JSX.Element => {
    const { appInfo } = useAppInfoCtx();
    const theme = useTheme();

    const gbansUrl = useMemo(() => {
        if (appInfo.app_version == 'master') {
            return 'https://github.com/leighmacdonald/gbans/tree/master';
        } else if (appInfo.app_version.startsWith('v')) {
            return `https://github.com/leighmacdonald/gbans/releases/tag/${appInfo.app_version}`;
        }
        return 'https://github.com/leighmacdonald/gbans';
    }, [appInfo.app_version]);

    return (
        <Box
            sx={{
                textAlign: 'center',
                marginTop: '1rem',
                padding: '1rem',
                marginBottom: '0',
                height: '100%'
            }}
        >
            <Grid container spacing={0} direction="column" alignItems="center" justifyContent="center">
                <Grid xs={3}>
                    <Typography variant={'subtitle2'} color={'text'}>
                        Copyright &copy; {appInfo.site_name} {new Date().getFullYear()}{' '}
                    </Typography>
                    <Stack
                        // direction={'row'}
                        alignItems="center"
                        justifyContent="center"
                    >
                        <Stack direction={'row'} spacing={1}>
                            <Link
                                component={RouterLink}
                                variant={'subtitle2'}
                                to={gbansUrl}
                                sx={{ color: theme.palette.text.primary }}
                            >
                                {appInfo.app_version}
                            </Link>
                            <Link
                                component={RouterLink}
                                variant={'subtitle2'}
                                to={'/changelog'}
                                sx={{ color: theme.palette.text.primary }}
                            >
                                Changelog
                            </Link>
                        </Stack>

                        <Link
                            component={RouterLink}
                            variant={'subtitle2'}
                            to={'/privacy-policy'}
                            sx={{ color: theme.palette.text.primary }}
                        >
                            Privacy Policy
                        </Link>
                    </Stack>
                </Grid>
            </Grid>
        </Box>
    );
};
