import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod/v4";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { logErr } from "../util/errors.ts";
import { once } from "../util/once.ts";

const exchangeToken = async (code: string): Promise<string> => {
	const res = await fetch("/api/auth/exchange", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ code }),
	});

	if (!res.ok) {
		throw new Error("exchange failed");
	}

	const body = (await res.json()) as { token: string };

	return body.token;
};

export const Route = createFileRoute("/_guest/login/success")({
	validateSearch: (search) => {
		// Backend sends snake_case query params, map to camelCase
		return z
			.object({
				nextUrl: z.string().optional().catch("/"),
				code: z.string().optional(),
			})
			.parse({
				...search,
				nextUrl: search.next_url ?? "/",
			});
	},
	component: LoginSteamSuccess,
	head: ({ match }) => ({
		meta: [match.context.title("Login Successful")],
	}),
});

function LoginSteamSuccess() {
	const search = Route.useSearch();
	const { login } = useAuth();
	const navigate = useNavigate();

	useEffect(() => {
		const code = search.code;
		if (!code) {
			navigate({ to: "/login", search: { redirect: "/" } });
			return;
		}

		let cancelled = false;

		// Exchange codes are single-use, so the exchange is dispatched only
		// once per code even if the effect runs more than once (StrictMode).
		once(code, async () => {
			const token = await exchangeToken(code);
			await login(token);
		}).then(
			() => {
				if (!cancelled) {
					navigate({ to: search.nextUrl });
				}
			},
			(error) => {
				if (cancelled) {
					return;
				}

				logErr(error);
				navigate({ to: "/" });
			},
		);

		return () => {
			cancelled = true;
		};
	}, [login, search.nextUrl, search.code, navigate]);

	return <LoadingPlaceholder />;
}
