import React, { useContext } from 'react';
import { noop } from 'lodash-es';

export const ColourModeContext = React.createContext({
    toggleColorMode: () => {
        noop();
    }
});

export const useColourModeCtx = () => useContext(ColourModeContext);
