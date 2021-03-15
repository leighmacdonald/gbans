import { Link } from 'react-router-dom';
import React, { forwardRef, useMemo } from 'react';
import { Button } from '@material-ui/core';

interface GLinkProps {
    icon?: string;
    primary: string;
    to: string;
}

export const GLink = ({ primary, to }: GLinkProps): JSX.Element => {
    const CustomLink = useMemo(() => {
        const f = forwardRef<HTMLAnchorElement>((linkProps, ref) => (
            <Link ref={ref} to={to} {...linkProps} />
        ));
        f.displayName = 'GLink';
        return f;
    }, [to]);
    return <Button component={CustomLink}>{primary}</Button>;
};
