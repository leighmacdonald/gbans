import { ElementType, PropsWithChildren } from 'react';
import Typography from '@mui/material/Typography';
// eslint-disable-next-line no-restricted-imports
import { Variant } from '@mui/material/styles/createTypography';
import { TableCellSmall } from './TableCellSmall.tsx';

interface TextProps {
    variant?: Variant;
    component?: ElementType;
}

export const TableCellString = ({ children, variant = 'body1', component = 'span' }: PropsWithChildren<TextProps>) => {
    return (
        <TableCellSmall>
            <Typography variant={variant} component={component}>
                {children}
            </Typography>
        </TableCellSmall>
    );
};
