import { createContext, useContext } from 'react';
import { ChatStatus, QueueMember, ChatLog, LobbyState } from '../api';
import { noop } from '../util/lists.ts';

type QueueCtxProps = {
    showChat: boolean;
    setShowChat: (showChat: boolean) => void;
    isReady: boolean;
    chatStatus: ChatStatus;
    reason: string;
    users: QueueMember[];
    lobbies: LobbyState[];
    messages: ChatLog[];
    joinQueue: (serverIds: string[]) => void;
    leaveQueue: (serverIds: string[]) => void;
    sendMessage: (message: string) => void;
};

export const QueueCtx = createContext<QueueCtxProps>({
    showChat: false,
    isReady: false,
    chatStatus: 'noaccess',
    reason: '',
    users: [],
    lobbies: [],
    messages: [],
    joinQueue: () => noop,
    leaveQueue: () => noop,
    sendMessage: () => noop,
    setShowChat: () => noop
});

export const useQueueCtx = () => useContext(QueueCtx);
