import React from "react";
import {PlayerBanForm} from "../component/PlayerBanForm";

export const AdminBan = () => {
    return (
        <div className="grid-container">
            <div className="grid-y grid-padding-y">
                <div className="cell">
                    <h1 className="text-center">Ban A Player Or Network</h1>
                </div>
                <PlayerBanForm />
            </div>
        </div>
    )
}