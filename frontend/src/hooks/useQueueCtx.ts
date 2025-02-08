import { createContext, useContext } from 'react';
import { ChatStatus, QueueMember, ServerQueueMessage, ServerQueueState } from '../api';
import { noop } from '../util/lists.ts';

type QueueCtxProps = {
    lastPong: Date;
    showChat: boolean;
    setShowChat: (showChat: boolean) => void;
    isReady: boolean;
    chatStatus: ChatStatus;
    reason: string;
    users: QueueMember[];
    servers: ServerQueueState[];
    messages: ServerQueueMessage[];
    joinQueue: (serverIds: string[]) => void;
    leaveQueue: (serverIds: string[]) => void;
    sendMessage: (message: string) => void;
};

export const QueueCtx = createContext<QueueCtxProps>({
    lastPong: new Date(),
    showChat: false,
    isReady: false,
    chatStatus: 'noaccess',
    reason: '',
    users: [],
    servers: [],
    messages: [],
    joinQueue: () => noop,
    leaveQueue: () => noop,
    sendMessage: () => noop,
    setShowChat: () => noop
});

export const useQueueCtx = () => useContext(QueueCtx);
