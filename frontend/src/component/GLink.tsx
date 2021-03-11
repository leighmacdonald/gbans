import { Link } from 'react-router-dom';
import React, {forwardRef, useMemo} from "react";
import {Button} from "@material-ui/core";

interface GLinkprops {
    icon?: string
    primary: string
    to: string
}
5
export const GLink = ({primary, to}: GLinkprops) => {
    const CustomLink = useMemo(
        () =>
            forwardRef<any>((linkProps, ref) => (
                <Link ref={ref} to={to} {...linkProps} />
            )),
        [to],
    );
    return (
        <Button component={CustomLink}>
            {primary}
        </Button>
    )
}

