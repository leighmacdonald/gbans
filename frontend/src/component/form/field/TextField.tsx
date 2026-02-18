import type { TextFieldProps } from "@mui/material/TextField";
import * as MUITextField from "@mui/material/TextField";
import { useStore } from "@tanstack/react-form";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import { renderHelpText } from "./renderHelpText.ts";

type Props = {
	label: string; // Make it required
} & TextFieldProps;

export const TextField = (props: Props) => {
	const field = useFieldContext<string>();
	const errors = useStore(field.store, (state) => state.meta.errors);

	return (
		<MUITextField.default
			{...props}
			fullWidth
			onChange={(e) => field.handleChange(e.target.value)}
			defaultValue={field.state.value}
			error={errors.length > 0}
			helperText={renderHelpText(errors, props.helperText)}
		/>
	);
};
