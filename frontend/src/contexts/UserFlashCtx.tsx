import { createContext } from 'react';
import { AlertColor } from '@mui/material/Alert';
import { Flash } from '../component/Flashes';
import { noop } from '../util/lists.ts';

export type CurrentFlashes = {
    flashes: Flash[];
    setFlashes: (flashes: Flash[]) => void;
    sendFlash: (level: AlertColor, message: string, heading?: string, closable?: boolean) => void;
    sendError: (error: unknown) => void;
};

export const UserFlashCtx = createContext<CurrentFlashes>({
    flashes: [],
    setFlashes: () => noop,
    sendFlash: () => noop,
    sendError: () => noop
});
