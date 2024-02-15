import { createContext, useContext } from 'react';
import { noop } from 'lodash-es';

export const ColourModeContext = createContext({
    toggleColorMode: () => {
        noop();
    }
});

export const useColourModeCtx = () => useContext(ColourModeContext);
