import { useContext } from 'react';
import { AuthContext } from '../auth.tsx';

export const useAuth = (): AuthContext => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};
