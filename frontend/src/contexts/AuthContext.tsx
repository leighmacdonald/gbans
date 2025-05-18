import { createContext } from 'react';
import { AuthContextProps } from '../auth.tsx';

export const AuthContext = createContext<AuthContextProps | null>(null);
