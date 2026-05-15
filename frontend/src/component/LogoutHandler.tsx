import { StorageKey } from "../auth.tsx";

export const LogoutHandler = () => {
	// Listen for storage events with the logout key and logout from all browser sessions/tabs when fired.
	window.addEventListener("storage", async (event) => {
		if (event.key === StorageKey.Logout) {
			localStorage.removeItem(StorageKey.Logout);
			document.location.reload();
		}
	});

	return null;
};
