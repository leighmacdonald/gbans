import { createContext } from 'react';
import { noop } from '../util/lists.ts';

export const ColourModeContext = createContext({
    toggleColorMode: () => {
        noop();
    }
});
