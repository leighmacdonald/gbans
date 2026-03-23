import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod/v4";
import { apiGetCurrentProfile } from "../api";
import { writeAccessToken } from "../util/auth/writeAccessToken.ts";
import { logErr } from "../util/errors.ts";

export const Route = createFileRoute("/_guest/login/success")({
	validateSearch: z.object({
		next_url: z.string().optional().catch(""),
		token: z.string(),
	}),
	component: LoginSteamSuccess,
	loaderDeps: ({ search }) => ({
		token: search.token,
	}),
	loader: async ({ context, deps }) => {
		const profile = await context.queryClient.fetchQuery({
			queryKey: ["currentUser"],
			queryFn: async ({ signal }) => {
				writeAccessToken(deps.token);
				return await apiGetCurrentProfile(signal);
			},
		});
		if (!profile) {
			throw "invalid profile";
		}
		context.auth?.login(profile);
		return profile;
	},
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const search = Route.useSearch();
	const navigate = useNavigate();

	navigate({ to: search.next_url }).catch(logErr);
}
