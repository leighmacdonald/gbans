import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod/v4";
import { StorageKey } from "../auth.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { useAuth } from "../hooks/useAuth.ts";

export const Route = createFileRoute("/_guest/login/success")({
	validateSearch: (search) => {
		// Backend sends snake_case query params, map to camelCase
		return z
			.object({
				nextUrl: z.string().optional().catch("/"),
				token: z.string().optional(),
			})
			.parse({
				...search,
				nextUrl: search.next_url ?? "/",
			});
	},
	component: LoginSteamSuccess,
	loaderDeps: ({ search }) => ({
		token: search.token,
	}),

	loader: ({ deps }) => {
		const savedToken = { token: deps.token };
		localStorage.setItem(StorageKey.Token, JSON.stringify(savedToken));
	},
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const { token } = Route.useLoaderDeps();
	const search = Route.useSearch();
	const { login } = useAuth();
	const navigate = useNavigate();

	useEffect(() => {
		if (!token) {
			navigate({ to: "/login", search: { redirect: "/" } });
			return;
		}

		let cancelled = false;

		const runLogin = async () => {
			try {
				await login(token, {
					onSuccess: async () => {
						if (cancelled) return;
						await navigate({ to: search.nextUrl });
					},
					onError: () => {
						if (cancelled) return;
						navigate({ to: "/" });
					},
				});
			} catch {
				if (cancelled) return;
				navigate({ to: "/" });
			}
		};

		runLogin();

		return () => {
			cancelled = true;
		};
	}, [login, search.nextUrl, token, navigate]);

	return <LoadingPlaceholder />;
}
