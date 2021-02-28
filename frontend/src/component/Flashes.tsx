import React from "react";

export interface Flash {
    level: string
    heading: string
    message: string
    closable?: boolean
    link_to?: string
}

export interface FlashesProps {
    flashes: Flash[]
}

export const Flashes = ({flashes}: FlashesProps): JSX.Element => {
    return (
        <>
            {flashes.map((f, i) => {
                return (<div className="cell" key={`flash-${i}`}>
                    <div className={`callout flash flash-${f.level}`}>
                        <h5>{f.heading}</h5>
                        <p>{f.message}</p>
                        {f.closable &&
                        <button className="close-button" aria-label="Dismiss alert" type="button">
                            <span aria-hidden="true"><i className={"fi-x"}/></span>
                        </button>
                        }
                    </div>
                </div>)
            })}
        </>
    )
}