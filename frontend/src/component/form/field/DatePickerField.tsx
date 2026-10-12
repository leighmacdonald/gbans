import { DesktopDatePicker, type DesktopDatePickerProps } from "@mui/x-date-pickers/DesktopDatePicker";
import { useSelector } from "@tanstack/react-form";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import { renderHelpText } from "./renderHelpText.ts";

type Props = { helpText?: string } & DesktopDatePickerProps;

export const DatePickerField = (props: Props) => {
	const field = useFieldContext<Date>();
	const errors = useSelector(field.store, (state) => state.meta.errors);
	const { helpText, ...pickerProps } = props;

	return (
		<DesktopDatePicker
			{...pickerProps}
			value={field.state.value}
			formatDensity={"spacious"}
			onChange={(value) => {
				field.handleChange(value as Date);
			}}
			slotProps={{
				textField: {
					fullWidth: true,
					variant: "filled",
					error: errors.length > 0,
					helperText: renderHelpText(errors, helpText),
				},
			}}
		/>
	);
};

export default DatePickerField;
