import { Link as RouterLink } from 'react-router-dom';
import Button from '@mui/material/Button';

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
