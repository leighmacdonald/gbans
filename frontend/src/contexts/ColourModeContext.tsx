import { createContext, useContext } from 'react';
import { noop } from '../util/lists.ts';

export const ColourModeContext = createContext({
    toggleColorMode: () => {
        noop();
    }
});

export const useColourModeCtx = () => useContext(ColourModeContext);
