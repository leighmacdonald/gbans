import Link from '@mui/material/Link';
import React from 'react';
import Container from '@mui/material/Container';

export const Footer = (): JSX.Element => {
    return (
        <Container
            sx={{
                textAlign: 'center',
                marginTop: '1rem',
                marginBottom: '1rem'
            }}
        >
            <Link
                sx={{
                    color: '#525252',
                    textDecoration: 'none'
                }}
                href={'https://github.com/leighmacdonald/gbans'}
            >
                gbans
            </Link>
        </Container>
    );
};
