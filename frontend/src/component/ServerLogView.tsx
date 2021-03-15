import React, { SyntheticEvent, useEffect, useState } from 'react';
import { includes, takeRight } from 'lodash-es';

interface ServerLog {
    CreatedOn: Date;
    ServerID: number;
    ServerName: string;
    Payload: Record<string, string | number | boolean>;
}

export const ServerLogView = (): JSX.Element => {
    const [serverIDs, setServerIDs] = useState<number[]>([]);
    const [entries, setEntries] = useState<ServerLog[]>([]);
    const [renderLimit, setRenderLimit] = useState<number>(10000);
    const [filterServerIDs, setFilterServerIDs] = useState<number[]>([]);
    useEffect(() => {
        setEntries([
            {
                CreatedOn: new Date(),
                Payload: {},
                ServerID: 1,
                ServerName: 'test-1'
            }
        ]);
    }, []);
    useEffect(() => {
        setServerIDs([1, 2]);
    }, []);
    useEffect(() => {
        setFilterServerIDs([1]);
    }, []);
    return (
        <>
            <div className={'grid-y grid-padding-y'}>
                <div className={'grid-x grid-padding-x'}>
                    <div className={'cell auto'}>
                        <select
                            onChange={(event: SyntheticEvent) => {
                                setRenderLimit(
                                    parseInt(
                                        (event.target as HTMLSelectElement)
                                            .value
                                    )
                                );
                            }}
                        >
                            <option value={100}>100</option>
                            <option value={500}>500</option>
                            <option value={1000}>1000</option>
                            <option value={10000}>10000</option>
                            <option value={Number.MAX_SAFE_INTEGER}>
                                inf.
                            </option>
                        </select>
                    </div>
                </div>
                {takeRight(
                    entries.filter((value) =>
                        filterServerIDs
                            ? includes(serverIDs, value.ServerID)
                            : false
                    ),
                    renderLimit
                ).map((value, i) => (
                    <div key={`log-${i}`}>
                        <div className={'cell'}>
                            <div>{value.Payload}</div>
                        </div>
                    </div>
                ))}
            </div>
        </>
    );
};
