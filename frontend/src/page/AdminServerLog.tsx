import React from 'react';
import {ServerLogView} from '../component/ServerLogView';

export const AdminServerLog = () => {
    return (
        <div className="grid-container">
            <div className="grid-y grid-padding-y">
                <div className="cell">
                    <h1 className="text-center">Game Server Logs</h1>
                </div>
                <ServerLogView />
            </div>
        </div>
    );
};
