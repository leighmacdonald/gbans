import Typography from '@mui/material/Typography';
import { Link as RouterLink } from '@tanstack/react-router';

interface ForumRowLinkProps {
    label: string;
    to: string;
    align?: 'inherit' | 'left' | 'center' | 'right' | 'justify';
    variant?: 'body2' | 'body1' | 'h6';
}

export const ForumRowLink = ({
    to,
    label,
    align,
    variant = 'h6'
}: ForumRowLinkProps) => {
    return (
        <Typography
            noWrap
            sx={{ textDecoration: 'none' }}
            fontWeight={700}
            width={'100%'}
            component={RouterLink}
            textOverflow={'ellipsis'}
            variant={variant}
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
