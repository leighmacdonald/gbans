import { StorageKey } from "../auth.tsx";
import { useAuth } from "../hooks/useAuth.ts";

export const LogoutHandler = () => {
	const { logout } = useAuth();

	// Listen for storage events with the logout key and logout from all browser sessions/tabs when fired.
	window.addEventListener("storage", async (event) => {
		if (event.key === StorageKey.Logout) {
			localStorage.removeItem(StorageKey.Logout);
			document.location.reload();
		}
	});

	logout().then(() => {});

	return null;
};
