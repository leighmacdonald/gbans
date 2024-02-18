import { useContext } from 'react';
import { ColourModeContext } from '../contexts/ColourModeContext.tsx';

export const useColourModeCtx = () => useContext(ColourModeContext);
