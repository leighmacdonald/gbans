import React from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';

export const Footer = (): JSX.Element => {
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
                    Copyright &copy; Uncletopia 2022
                </Typography>
                {/*<Link*/}
                {/*    sx={{*/}
                {/*        color: '#525252',*/}
                {/*        textDecoration: 'none'*/}
                {/*    }}*/}
                {/*    href={'https://github.com/leighmacdonald/gbans'}*/}
                {/*>*/}
                {/*    gbans*/}
                {/*</Link>*/}
            </Stack>
        </Box>
    );
};
