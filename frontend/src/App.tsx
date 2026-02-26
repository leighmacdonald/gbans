import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type AnyRouter, RouterProvider } from "@tanstack/react-router";
import { StrictMode, useState } from "react";
import { defaultAvatarHash } from "./api";
import { AuthProvider, profileKey } from "./auth.tsx";
import { useAuth } from "./hooks/useAuth.ts";
import { PermissionLevel } from "./schema/people.ts";
import { logErr } from "./util/errors.ts";

const loadProfile = () => {
	const defaultProfile = {
		steam_id: "",
		permission_level: PermissionLevel.Guest,
		avatarhash: defaultAvatarHash,
		name: "",
		ban_id: 0,
		muted: false,
		discord_id: "",
		created_on: new Date(),
		updated_on: new Date(),
	};
	try {
		const userData = localStorage.getItem(profileKey);
		if (!userData) {
			return defaultProfile;
		}

		return JSON.parse(userData);
	} catch (e) {
		logErr(e);
		return defaultProfile;
	}
};

export function App({ queryClient, router }: { queryClient: QueryClient; router: AnyRouter }) {
	const [profile, setProfile] = useState(loadProfile());

	return (
		<AuthProvider profile={profile} setProfile={setProfile}>
			<QueryClientProvider client={queryClient}>
				<StrictMode>
					<InnerApp router={router} />
				</StrictMode>
			</QueryClientProvider>
		</AuthProvider>
	);
}

const InnerApp = ({ router }: { router: AnyRouter }) => {
	const auth = useAuth();

	return <RouterProvider defaultPreload={"intent"} router={router} context={{ auth }} />;
};
