import { JSX, useMemo } from 'react';
import ModalImage from 'react-modal-image';
import { PaletteMode } from '@mui/material';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { getOverrides, MuiMarkdown } from 'mui-markdown';
import { Highlight, themes } from 'prism-react-renderer';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';
import RouterLink from './RouterLink.tsx';

const renderLinks = (body_md: string, asset_url: string): string => {
    return body_md
        .replace('/^[\u200B\u200C\u200D\u200E\u200F\uFEFF]/', '')
        .replace(/(wiki:\/\/)/gi, '/wiki/')
        .replace(/(media:\/\/)/gi, asset_url != '' ? asset_url : '/asset/' + '/');
};

interface MDImgProps {
    children: JSX.Element;
    src: string;
    alt: string;
    title: string;
}

const MDImg = ({ src, alt }: MDImgProps) => {
    return <ModalImage small={src} large={src} alt={alt} />;
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
                color: (theme) => theme.palette.text.primary,
                textDecoration: 'none',
                '&:hover': { textDecoration: 'underline' }
            }}
        >
            {children}
        </Typography>
    );
};

export const MarkDownRenderer = ({ body_md, minHeight }: { body_md: string; minHeight?: number }) => {
    const theme = (localStorage.getItem('theme') as PaletteMode) || 'dark';
    const { appInfo } = useAppInfoCtx();

    const links = useMemo(() => {
        return renderLinks(body_md, appInfo.asset_url);
    }, [appInfo.asset_url, body_md]);

    return (
        <Box padding={2} maxWidth={'100%'} minHeight={minHeight}>
            <MuiMarkdown
                options={{
                    disableParsingRawHTML: false,
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
                {links}
            </MuiMarkdown>
        </Box>
    );
};
