import { createContext, useContext } from 'react';
import { Flash } from '../component/Flashes';
import { noop } from 'lodash-es';
import { AlertColor } from '@mui/material/Alert';

export type CurrentFlashes = {
    flashes: Flash[];
    setFlashes: (flashes: Flash[]) => void;
    sendFlash: (
        level: AlertColor,
        message: string,
        heading?: string,
        closable?: boolean
    ) => void;
};
export const UserFlashCtx = createContext<CurrentFlashes>({
    flashes: [],
    setFlashes: () => noop,
    sendFlash: () => noop
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useUserFlashCtx = () => useContext(UserFlashCtx);
