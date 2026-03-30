import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { type JSX, useMemo } from "react";
import type { InfoResponse } from "../gen/config/v1/config_pb.ts";
import RouterLink from "./RouterLink.tsx";

export const Footer = ({ appInfo }: { appInfo: InfoResponse }): JSX.Element => {
	const theme = useTheme();

	const gbansUrl = useMemo(() => {
		if (appInfo.appVersion === "master") {
			return "https://github.com/leighmacdonald/gbans/tree/master";
		} else if (appInfo.appVersion.startsWith("v")) {
			return `https://github.com/leighmacdonald/gbans/releases/tag/${appInfo.appVersion}`;
		}
		return "https://github.com/leighmacdonald/gbans";
	}, [appInfo.appVersion]);

	return (
		<Box
			sx={{
				textAlign: "center",
				marginTop: "1rem",
				padding: "1rem",
				marginBottom: "0",
				height: "100%",
			}}
		>
			<Grid container spacing={0} direction="column" alignItems="center" justifyContent="center">
				<Grid size={{ xs: 3 }}>
					<Typography variant={"subtitle2"} color={"text"}>
						Copyright &copy; {appInfo.siteName} {new Date().getFullYear()}{" "}
					</Typography>
					<Stack
						// direction={'row'}
						alignItems="center"
						justifyContent="center"
					>
						<Stack direction={"row"} spacing={1}>
							<Link
								component={RouterLink}
								variant={"subtitle2"}
								to={gbansUrl}
								sx={{ color: theme.palette.text.primary }}
							>
								{appInfo.appVersion}
							</Link>
							<Link
								component={RouterLink}
								variant={"subtitle2"}
								to={"/changelog"}
								sx={{ color: theme.palette.text.primary }}
							>
								Changelog
							</Link>
						</Stack>

						<Link
							component={RouterLink}
							variant={"subtitle2"}
							to={"/privacy-policy"}
							sx={{ color: theme.palette.text.primary }}
						>
							Privacy Policy
						</Link>
					</Stack>
				</Grid>
			</Grid>
		</Box>
	);
};
