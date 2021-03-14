import {createContext, useContext} from 'react';
import {Flash} from '../component/Flashes';

export type CurrentFlashes = {
    flashes: Flash[];
    setFlashes?: (f: Flash[]) => void;
};
export const UserFlashCtx = createContext<CurrentFlashes>({
    flashes: []
});

export const useUserFlashCtx = () => useContext(UserFlashCtx);
