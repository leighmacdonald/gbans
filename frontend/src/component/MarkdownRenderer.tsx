import React, { JSX } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { getOverrides, MuiMarkdown } from 'mui-markdown';
import { Highlight, themes } from 'prism-react-renderer';

export const renderLinks = (body_md: string): string => {
    return body_md
        .replace('/^[\u200B\u200C\u200D\u200E\u200F\uFEFF]/', '')
        .replace(/(wiki:\/\/)/gi, '/wiki/')
        .replace(
            /(media:\/\/)/gi,
            window.gbans.asset_url + '/' + window.gbans.bucket_media + '/'
        );
};

interface MDImgProps {
    children: JSX.Element;
    src: string;
    alt: string;
    title: string;
}

const MDImg = ({ src, alt, title }: MDImgProps) => {
    return (
        <a href={src}>
            <img src={src} alt={alt} title={title} className={'img_media'} />
        </a>
    );
};

interface MDLnkProps {
    children: JSX.Element;
    href: string;
    title: string;
}

const MDLink = ({ children, href, title }: MDLnkProps) => {
    return (
        <Typography
            variant={'body1'}
            component={RouterLink}
            to={href}
            title={title}
            fontWeight={700}
            sx={{
                textDecoration: 'none',
                '&:hover': { textDecoration: 'underline' }
            }}
            color={(theme) => theme.palette.text.primary}
        >
            {children}
        </Typography>
    );
};

export const MarkDownRenderer = ({ body_md }: { body_md: string }) => {
    return (
        <Box padding={2} maxWidth={'100%'}>
            <MuiMarkdown
                options={{
                    overrides: {
                        ...getOverrides({
                            Highlight,
                            themes,
                            theme: themes.github
                        }),
                        a: {
                            component: MDLink
                        },
                        img: {
                            component: MDImg
                        }
                    }
                }}
                prismTheme={themes.github}
            >
                {renderLinks(body_md)}
            </MuiMarkdown>
        </Box>
    );
};
