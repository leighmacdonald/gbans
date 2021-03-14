import {Link} from 'react-router-dom';
import React, {forwardRef, useMemo} from 'react';
import {Button} from '@material-ui/core';

interface GLinkProps {
    icon?: string;
    primary: string;
    to: string;
}

export const GLink = ({primary, to}: GLinkProps) => {
    const CustomLink = useMemo(
        () => forwardRef<any>((linkProps, ref) => <Link ref={ref} to={to} {...linkProps} />),
        [to]
    );
    return <Button component={CustomLink}>{primary}</Button>;
};
