import type { TextFieldProps } from "@mui/material/TextField";
import * as MUITextField from "@mui/material/TextField";
import { useSelector } from "@tanstack/react-form";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import { renderHelpText } from "./renderHelpText.ts";

type Props = {
	label: string; // Make it required
} & TextFieldProps;

export const TextField = (props: Props) => {
	const field = useFieldContext<string>();
	const errors = useSelector(field.store, (state) => state.meta.errors);

	// Must stay controlled (value, not defaultValue): an uncontrolled input only
	// seeds its DOM value on mount, so if the form instance is swapped while the
	// component stays mounted (e.g. re-showing a NiceModal with different
	// defaultValues) the stale value stays on screen.
	return (
		<MUITextField.default
			{...props}
			fullWidth
			onChange={(e) => field.handleChange(e.target.value)}
			value={field.state.value ?? ""}
			error={errors.length > 0}
			helperText={renderHelpText(errors, props.helperText)}
		/>
	);
};

export default TextField;
