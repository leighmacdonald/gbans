import { ElementType, PropsWithChildren } from 'react';
import Typography from '@mui/material/Typography';
// eslint-disable-next-line no-restricted-imports
import { Variant } from '@mui/material/styles/createTypography';

interface TextProps {
    variant?: Variant;
    component?: ElementType;
    onClick?: () => void;
}

export const TableCellString = ({ children, variant = 'body1', component = 'p' }: PropsWithChildren<TextProps>) => {
    return (
        <Typography variant={variant} component={component}>
            {children}
        </Typography>
    );
};
