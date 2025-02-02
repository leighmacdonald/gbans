import { useEffect, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { isOperationType, Operation, pingPayload, websocketURL } from '../api';

export const WebsocketConnector = () => {
    const [isReady, setIsReady] = useState(false);
    const [lastPong, setLastPong] = useState(new Date());

    const { readyState, sendJsonMessage, lastJsonMessage } = useWebSocket(websocketURL(), {
        filter: (message) => {
            if (isOperationType(message, Operation.Pong)) {
                setLastPong(new Date());

                return false;
            }
            return true;
        },
        share: true
    });

    useEffect(() => {}, [lastJsonMessage]);

    useEffect(() => {
        switch (readyState) {
            case ReadyState.OPEN:
                setIsReady(true);
                sendJsonMessage({ op: Operation.Ping, payload: { created_on: new Date() } } as pingPayload);
                break;
            default:
                setIsReady(false);
        }
        console.log(`readyState: ${readyState} lastPong: ${lastPong} isReady: ${isReady}`);
    }, [readyState]);

    return <></>;
};
