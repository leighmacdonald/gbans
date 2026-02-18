import Button, { type ButtonProps } from "@mui/material/Button";
import { styled } from "@mui/material/styles";

export const LeftAlignButton = styled(Button)<ButtonProps>(() => ({
	justifyContent: "space-between",
}));
