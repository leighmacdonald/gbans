import { ElementType, PropsWithChildren } from 'react';
import { TypographyVariant } from '@mui/material';
import Typography from '@mui/material/Typography';

interface TextProps {
    variant?: TypographyVariant;
    component?: ElementType;
    onClick?: () => void;
}

export const TableCellString = ({ children, variant = 'body1', component = 'p' }: PropsWithChildren<TextProps>) => {
    return (
        <div>
            <Typography variant={variant} component={component}>
                {children}
            </Typography>
        </div>
    );
};
