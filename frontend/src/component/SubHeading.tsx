import { PropsWithChildren } from 'react';
import { Typography } from '@mui/material';

export const SubHeading = ({ children }: PropsWithChildren) => (
    <Typography variant={'subtitle1'} padding={1}>
        {children}
    </Typography>
);
