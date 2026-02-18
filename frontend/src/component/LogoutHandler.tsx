import { logoutKey } from "../auth.tsx";

export const LogoutHandler = () => {
	// Listen for storage events with the logout key and logout from all browser sessions/tabs when fired.
	window.addEventListener("storage", async (event) => {
		if (event.key === logoutKey) {
			localStorage.removeItem(logoutKey);
			document.location.reload();
		}
	});

	return <></>;
};
