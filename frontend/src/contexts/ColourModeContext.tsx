import { noop } from 'lodash-es';
import React, { useContext } from 'react';

export const ColourModeContext = React.createContext({
    toggleColorMode: () => {
        noop();
    }
});

export const useColourModeCtx = () => useContext(ColourModeContext);
