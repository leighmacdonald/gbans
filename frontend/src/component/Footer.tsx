import React, { JSX, useMemo } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import { Link as RouterLink } from 'react-router-dom';
import { useTheme } from '@mui/material/styles';

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
            <Stack>
                <Typography variant={'subtitle2'} color={'text'}>
                    Copyright &copy; {window.gbans.site_name || 'gbans'}{' '}
                    {new Date().getFullYear()}{' '}
                </Typography>
                <Link
                    component={RouterLink}
                    variant={'subtitle2'}
                    to={gbansUrl}
                    sx={{ color: theme.palette.text.primary }}
                >
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
        </Box>
    );
};
