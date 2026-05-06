import { useQuery } from "@connectrpc/connect-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod/v4";
import { StorageKey } from "../auth.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { currentProfile } from "../rpc/person/v1/person-PersonService_connectquery.ts";
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
		const savedToken = { token: deps.token };
		localStorage.setItem(StorageKey.Token, JSON.stringify(savedToken));
	},
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const search = Route.useSearch();

	const navigate = useNavigate();
	const { login } = useAuth();
	const { data } = useQuery(currentProfile, {});

	useEffect(() => {
		if (data?.profile) {
			login(data.profile, search.token);
			navigate({ to: !emptyOrNullString(search.next_url) ? search.next_url : "/" });
		}
	}, [data?.profile, login, navigate, search.next_url, search.token]);

	return <LoadingPlaceholder />;
}
