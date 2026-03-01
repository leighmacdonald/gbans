import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useEffect, useLayoutEffect } from "react";
import { z } from "zod/v4";
import { apiGetCurrentProfile } from "../api";
import { useAuth } from "../hooks/useAuth.ts";
import { writeAccessToken } from "../util/auth/writeAccessToken.ts";

export const Route = createFileRoute("/_guest/login/success")({
	validateSearch: z.object({
		next_url: z.string().optional().catch(""),
		token: z.string(),
	}),
	component: LoginSteamSuccess,
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const router = useRouter();
	const { login } = useAuth();
	const search = Route.useSearch();

	const { data: profile } = useQuery({
		queryKey: ["currentUser"],
		queryFn: async () => {
			writeAccessToken(search.token);
			return await apiGetCurrentProfile();
		},
	});

	useEffect(() => {
		if (!profile) {
			return;
		}

		login(profile);
		router.invalidate();
	}, [login, profile, router]);

	useLayoutEffect(() => {
		if (!profile) {
			return;
		}

		if (profile.steam_id !== "" && search.next_url) {
			router.history.push(search.next_url);
		}
	}, [profile, router.history, search, search.next_url]);

	return <>{<Typography variant={"h3"}>Logging In...</Typography>}</>;
}
