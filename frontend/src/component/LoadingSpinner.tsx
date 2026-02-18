import Button from "@mui/material/Button";
import { useTheme } from "@mui/material/styles";
import { LoadingIcon } from "./LoadingIcon";

export const LoadingSpinner = () => {
	const theme = useTheme();
	return (
		<Button
			title={"Loading..."}
			loading
			loadingIndicator={<LoadingIcon />}
			variant={"text"}
			color={"secondary"}
			sx={{ color: theme.palette.text.primary }}
		>
			Loading...
		</Button>
	);
};
