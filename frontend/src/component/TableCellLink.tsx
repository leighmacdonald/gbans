import Button from '@mui/material/Button';
import RouterLink from './RouterLink.tsx';

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
