import Typography from "@mui/material/Typography";
import RouterLink from "../RouterLink.tsx";

interface ForumRowLinkProps {
	label: string;
	to: string;
	align?: "inherit" | "left" | "center" | "right" | "justify";
	variant?: "body2" | "body1" | "h6";
}

export const ForumRowLink = ({ to, label, align, variant = "h6" }: ForumRowLinkProps) => {
	return (
        <Typography
            noWrap
            component={RouterLink}
            variant={variant}
            to={to}
            align={align}
            sx={{
                fontWeight: 700,
                width: "100%",
                textOverflow: "ellipsis",
                color: (theme) => theme.palette.text.primary,
                textDecoration: "none"
            }}>
            {label}
        </Typography>
    );
};
