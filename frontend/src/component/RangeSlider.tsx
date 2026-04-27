import { Slider } from "@mui/material";
import { styled } from "@mui/material/styles";

export const RangeSlider = styled(Slider)(({ theme }) => ({
	height: 2,
	padding: "15px 0",
	"& .MuiSlider-thumb": {
		backgroundColor: theme.palette.common.white,
	},
	"& .MuiSlider-valueLabel": {
		color: theme.palette.common.white,
		"&:before": {
			display: "none",
		},
		"& *": {
			background: "transparent",
			color: theme.palette.common.white,
		},
	},
	"& .MuiSlider-track": {
		border: "none",
	},
	"& .MuiSlider-rail": {
		opacity: 0.5,
		backgroundColor: "#bfbfbf",
	},
	"& .MuiSlider-mark": {
		backgroundColor: "#bfbfbf",
		height: 8,
		width: 1,
		"&.MuiSlider-markActive": {
			opacity: 1,
			backgroundColor: "currentColor",
		},
	},
}));
