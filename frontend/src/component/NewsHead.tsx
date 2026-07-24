import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import Stack from "@mui/system/Stack";
import type { ReactNode } from "react";
import { tf2Fonts } from "../theme";

export const NewsHead = ({ left, right }: { left: ReactNode; right: ReactNode }) => {
	const theme = useTheme();
	return (
        <Stack
            spacing={1}
            direction={"row"}
            sx={{
                padding: 1,
                paddingLeft: 2,
                paddingRight: 2,
                backgroundColor: theme.palette.primary.main
            }}>
            <Typography
                variant={"subtitle1"}
                align={"left"}
                sx={{
                    lineHeight: 2,
                    color: theme.palette.primary.contrastText,
                    width: "100%",
                    ...tf2Fonts
                }}>
				{left}
			</Typography>
            <Typography
                variant={"subtitle1"}
                align={"right"}
                sx={{
                    lineHeight: 2,
                    ...tf2Fonts,
                    color: theme.palette.primary.contrastText
                }}>
				{right}
			</Typography>
        </Stack>
    );
};
