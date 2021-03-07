import React from "react";

export const Footer = () => {
    return (
        <footer className="grid-container full" style={{"marginTop": "10em"}}>
            <div className="grid-x grid-padding-x" id="footer">
                <div className="cell">
                    <p>Powered By: <a href="https://github.com/leighmacdonald/gbans">gbans</a></p>
                </div>
            </div>
        </footer>
    )
}