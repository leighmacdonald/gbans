import { createContext, useContext } from 'react';
import { Flash } from '../component/Flashes';

export type CurrentFlashes = {
    flashes: Flash[];
    setFlashes: (f: Flash[]) => void;
};
export const UserFlashCtx = createContext<CurrentFlashes>({
    flashes: [],
    setFlashes: (_) => {}
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useUserFlashCtx = () => useContext(UserFlashCtx);
