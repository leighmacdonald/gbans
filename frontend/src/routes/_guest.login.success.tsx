import { useQuery } from "@connectrpc/connect-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod/v4";
import { accessTokenKey } from "../auth.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { currentProfile } from "../rpc/person/v1/person-PersonService_connectquery.ts";
import { logErr } from "../util/errors.ts";
import { emptyOrNullString } from "../util/types.ts";

export const Route = createFileRoute("/_guest/login/success")({
	validateSearch: z.object({
		next_url: z.string().optional().catch(""),
		token: z.string(),
	}),
	component: LoginSteamSuccess,
	loaderDeps: ({ search }) => ({
		token: search.token,
	}),
	loader: async ({ deps }) => {
		localStorage.setItem(accessTokenKey, deps.token);
	},
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const { login } = useAuth();

	const { data, isLoading } = useQuery(currentProfile);

	if (isLoading || !data?.profile) {
		return <LoadingPlaceholder />;
	}

	login(data.profile);

	if (!emptyOrNullString(search.next_url)) {
		navigate({ to: search.next_url }).catch(logErr);
	} else {
		navigate({ to: "/" }).catch(logErr);
	}
}
