import { JSX, useMemo } from 'react';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { Link as RouterLink } from '@tanstack/react-router';

export const Footer = (): JSX.Element => {
    const theme = useTheme();

    const gbansUrl = useMemo(() => {
        if (window.gbans.build_version == 'master') {
            return 'https://github.com/leighmacdonald/gbans/tree/master';
        } else if (window.gbans.build_version.startsWith('v')) {
            return `https://github.com/leighmacdonald/gbans/releases/tag/${window.gbans.build_version}`;
        }
        return 'https://github.com/leighmacdonald/gbans';
    }, []);

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
                        Copyright &copy; {window.gbans.site_name || 'gbans'} {new Date().getFullYear()}{' '}
                    </Typography>
                    <Stack
                        // direction={'row'}
                        alignItems="center"
                        justifyContent="center"
                    >
                        <Link component={RouterLink} variant={'subtitle2'} to={gbansUrl} sx={{ color: theme.palette.text.primary }}>
                            {window.gbans.build_version}
                        </Link>
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
