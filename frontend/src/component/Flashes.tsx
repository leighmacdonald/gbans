import React from "react";
import {Alert} from "@material-ui/lab";
import {Color} from "@material-ui/lab/Alert/Alert";

export interface Flash {
    level: Color
    heading: string
    message: string
    closable?: boolean
    link_to?: string
}

export interface FlashesProps {
    flashes: Flash[]
}

export const Flashes = ({flashes}: FlashesProps): JSX.Element =>
    <>
        {flashes.map((f) => {
            return (<Alert severity={f.level}>{f.message}</Alert>)
        })}
    </>
