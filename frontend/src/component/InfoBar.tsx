import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

interface InfoBarProps {
	title: string;
	value: string | number;
	align?: "left" | "right";
}

export const InfoBar = ({ title, value, align = "left" }: InfoBarProps) => {
	return (
        <Box>
            <Typography variant={"subtitle1"} align={align} sx={{
                fontWeight: 500
            }}>
				{title}
			</Typography>
            <Typography variant={"h3"} align={align} sx={{
                fontWeight: 700
            }}>
				{value}
			</Typography>
        </Box>
    );
};
