import Box from "@mui/material/Box";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Typography from "@mui/material/Typography";
import { useFieldContext } from "../../../contexts/formContext.tsx";

type Props = {
	readonly label?: string;
	readonly disabled?: boolean;
};

// Tri-state boolean options: on, off, or default (no override, the server
// decides). The field value is the option key ("on" | "default" | "off").
const options = [
	{ value: "on", label: "On" },
	{ value: "default", label: "Default" },
	{ value: "off", label: "Off" },
] as const;

export const TriStateField = ({ label, disabled = false }: Props) => {
	const field = useFieldContext<string>();

	return (
		<Box
			sx={{
				justifyContent: "flex-start",
				alignItems: "center",
				gap: 2,
				marginTop: 1,
			}}
		>
			{label ? <Typography>{label}</Typography> : null}
			<ToggleButtonGroup
				size={"small"}
				exclusive
				disabled={disabled}
				value={field.state.value}
				onChange={(_, value) => {
					if (value !== null) {
						field.handleChange(value);
					}
				}}
				aria-label={label}
			>
				{options.map((o) => {
					return (
						<ToggleButton key={o.value} value={o.value} onBlur={field.handleBlur}>
							{o.label}
						</ToggleButton>
					);
				})}
			</ToggleButtonGroup>
		</Box>
	);
};

export default TriStateField;
