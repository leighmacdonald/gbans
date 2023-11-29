import React from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Typography from '@mui/material/Typography';

interface ForumRowLinkProps {
    label: string;
    to: string;
    align?: 'inherit' | 'left' | 'center' | 'right' | 'justify';
}

export const ForumRowLink = ({ to, label, align }: ForumRowLinkProps) => {
    return (
        <Typography
            component={RouterLink}
            variant={'h6'}
            to={to}
            align={align}
            color={(theme) => {
                return theme.palette.text.primary;
            }}
        >
            {label}
        </Typography>
    );
};
