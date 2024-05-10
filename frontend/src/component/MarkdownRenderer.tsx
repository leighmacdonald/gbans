import { JSX } from 'react';
import { PaletteMode } from '@mui/material';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { getOverrides, MuiMarkdown } from 'mui-markdown';
import { Highlight, themes } from 'prism-react-renderer';
import RouterLink from './RouterLink.tsx';

const renderLinks = (body_md: string): string => {
    return body_md
        .replace('/^[\u200B\u200C\u200D\u200E\u200F\uFEFF]/', '')
        .replace(/(wiki:\/\/)/gi, '/wiki/')
        .replace(/(media:\/\/)/gi, window.gbans.asset_url + '/' + window.gbans.bucket_media + '/');
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

export const MarkDownRenderer = ({ body_md, minHeight }: { body_md: string; minHeight?: number }) => {
    const theme = (localStorage.getItem('theme') as PaletteMode) || 'dark';

    return (
        <Box padding={2} maxWidth={'100%'} minHeight={minHeight}>
            <MuiMarkdown
                options={{
                    overrides: {
                        ...getOverrides({
                            Highlight,
                            themes,
                            theme: theme == 'dark' ? themes.vsDark : themes.vsLight
                        }),
                        a: {
                            component: MDLink
                        },
                        img: {
                            component: MDImg
                        },
                        // p: {
                        //     props: {
                        //         gutterBottom: true
                        //     }
                        // },
                        h1: {
                            props: {
                                variant: 'h3'
                            }
                        },
                        h2: {
                            props: {
                                variant: 'h3'
                            }
                        },
                        h3: {
                            props: {
                                variant: 'h3'
                            }
                        }
                    }
                }}
            >
                {renderLinks(body_md)}
            </MuiMarkdown>
        </Box>
    );
};
