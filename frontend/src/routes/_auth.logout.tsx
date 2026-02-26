import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAuth } from "../hooks/useAuth";

export const Route = createFileRoute("/_auth/logout")({
	component: LogoutPage,
	loader: ({ context }) => ({
		appInfo: context.appInfo,
	}),
	head: ({ loaderData }) => ({
		meta: [{ title: `Logout - ${loaderData?.appInfo.site_name}` }],
	}),
});

function LogoutPage() {
	const navigate = useNavigate();
	const { logout } = useAuth();

	useEffect(() => {
		logout();
		navigate({ to: "/" });
	}, [logout, navigate]);

	return <Typography variant={"h2"}>Logging out...</Typography>;
}
