import Box from '@mui/material/Box';
import React from 'react';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import EditIcon from '@mui/icons-material/Edit';

export interface MarkdownBoxProps {
    bodyHTML: string;
    readonly: boolean;
    setEditMode: (mode: boolean) => void;
}

export const RenderedMarkdownBox = ({
    bodyHTML,
    readonly,
    setEditMode
}: MarkdownBoxProps) => {
    return (
        <Box padding={2}>
            <Box
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
                dangerouslySetInnerHTML={{ __html: bodyHTML }}
            />
            {!readonly && (
                <ButtonGroup>
                    <Button
                        variant={'contained'}
                        color={'primary'}
                        onClick={() => {
                            setEditMode(true);
                        }}
                        startIcon={<EditIcon />}
                    >
                        Edit Page
                    </Button>
                </ButtonGroup>
            )}
        </Box>
    );
};
