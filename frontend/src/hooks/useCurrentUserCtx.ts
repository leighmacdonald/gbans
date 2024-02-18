import { useContext } from 'react';
import { CurrentUserCtx } from '../contexts/CurrentUserCtx.tsx';

export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
