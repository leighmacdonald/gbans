import Box from "@mui/material/Box";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormGroup from "@mui/material/FormGroup";
import { useFieldContext } from "../../../contexts/formContext.tsx";

type Props = {
	readonly disabled?: boolean;
	readonly label?: string;
};

export const CheckboxField = ({ label, disabled = false }: Props) => {
	const field = useFieldContext<boolean>();

	return (
		<Box display="flex" justifyContent="flex-start" alignItems="center" marginTop={1}>
			<FormGroup>
				<FormControlLabel
					control={
						<Checkbox
							onChange={(_, v) => {
								field.handleChange(v);
							}}
							onBlur={field.handleBlur}
							checked={Boolean(field.state.value)}
							name={field.name}
							disabled={disabled}
						/>
					}
					label={label}
				/>
			</FormGroup>
		</Box>
	);
};
