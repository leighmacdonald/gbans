import React from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Typography from '@mui/material/Typography';

interface ForumRowLinkProps {
    label: string;
    to: string;
}

export const ForumRowLink = ({ to, label }: ForumRowLinkProps) => {
    return (
        <Typography
            component={RouterLink}
            variant={'h5'}
            to={to}
            color={(theme) => {
                return theme.palette.text.primary;
            }}
        >
            {label}
        </Typography>
    );
};
