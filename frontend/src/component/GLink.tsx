import { Link } from 'react-router-dom';
import React, { forwardRef, useMemo } from 'react';
import { Button } from '@mui/material';

interface GLinkProps {
    icon?: string | JSX.Element;
    primary: string;
    to: string;
}

export const GLink = ({ primary, to, icon }: GLinkProps): JSX.Element => {
    const CustomLink = useMemo(() => {
        const f = forwardRef<HTMLAnchorElement>((linkProps, ref) => (
            <Link ref={ref} to={to} {...linkProps} />
        ));
        f.displayName = 'GLink';
        return f;
    }, [to]);
    return (
        <Button component={CustomLink} startIcon={icon}>
            {primary}
        </Button>
    );
};
