import Box from '@mui/material/Box';
import React from 'react';

export interface MarkdownBoxProps {
    bodyMd: string;
}

export const RenderedMarkdownBox = ({ bodyMd }: MarkdownBoxProps) => {
    return (
        <Box
            padding={2}
            sx={(theme) => {
                return {
                    img: {
                        maxWidth: '100%'
                    },
                    a: {
                        color: theme.palette.text.primary
                    }
                };
            }}
            dangerouslySetInnerHTML={{ __html: bodyMd }}
        />
    );
};
