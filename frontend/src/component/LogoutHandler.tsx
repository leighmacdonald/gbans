import { logoutKey } from '../auth.tsx';
import { writeAccessToken } from '../util/auth/writeAccessToken.ts';
import { writeRefreshToken } from '../util/auth/writeRefreshToken.ts';

export const LogoutHandler = () => {
    // Listen for storage events with the logout key and logout from all browser sessions/tabs when fired.
    window.addEventListener('storage', async (event) => {
        if (event.key === logoutKey) {
            localStorage.removeItem(logoutKey);
            writeAccessToken('');
            writeRefreshToken('');
            document.location.reload();
        }
    });

    return <></>;
};
