import Button from '@mui/material/Button';
import { Link as RouterLink } from '@tanstack/react-router';

interface TableCellLinkProps {
    label: string;
    to: string;
}

export const TableCellLink = ({ to, label }: TableCellLinkProps) => {
    return (
        <Button fullWidth component={RouterLink} variant={'text'} to={to}>
            {label}
        </Button>
    );
};
