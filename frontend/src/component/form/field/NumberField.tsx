import { NumberField as BaseNumberField } from "@base-ui/react/number-field";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import KeyboardArrowUpIcon from "@mui/icons-material/KeyboardArrowUp";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import InputLabel from "@mui/material/InputLabel";
import OutlinedInput from "@mui/material/OutlinedInput";
import { useStore } from "@tanstack/react-form";
import { useId } from "react";
import { useFieldContext } from "../../../contexts/formContext";

type Props = {
	label: string; // Make it required
} & BaseNumberField.Root.Props;

export const NumberField = ({ label, id: idProp, min, max }: Props) => {
	const field = useFieldContext<number>();
	const errors = useStore(field.store, (state) => state.meta.errors);

	let id = useId();
	if (idProp) {
		id = idProp;
	}

	console.log(field.state.value);
	return (
		<BaseNumberField.Root
			render={(props, state) => (
				<FormControl
					size={"small"}
					ref={props.ref}
					disabled={state.disabled}
					required={state.required}
					error={errors.length > 0}
					variant="outlined"
				>
					{props.children}
				</FormControl>
			)}
		>
			<InputLabel htmlFor={id}>{label}</InputLabel>
			<BaseNumberField.Input
				id={id}
				render={(props) => (
					<OutlinedInput
						label={label}
						// defaultValue={field.state.value}
						value={field.state.value}
						onChange={(e) => field.handleChange(Number(e.target.value))}
						onKeyUp={props.onKeyUp}
						onKeyDown={props.onKeyDown}
						onFocus={props.onFocus}
						// slotProps={{
						// 	input: props,
						// }}
						// endAdornment={
						// 	<InputAdornment
						// 		position="end"
						// 		sx={{
						// 			flexDirection: "column",
						// 			maxHeight: "unset",
						// 			alignSelf: "stretch",
						// 			borderLeft: "1px solid",
						// 			borderColor: "divider",
						// 			ml: 0,
						// 			"& button": {
						// 				py: 0,
						// 				flex: 1,
						// 				borderRadius: 0.5,
						// 			},
						// 		}}
						// 	>
						// 		<BaseNumberField.Increment render={<IconButton size={"small"} aria-label="Increase" />}>
						// 			<KeyboardArrowUpIcon fontSize={"small"} sx={{ transform: "translateY(2px)" }} />
						// 		</BaseNumberField.Increment>

						// 		<BaseNumberField.Decrement render={<IconButton size={"small"} aria-label="Decrease" />}>
						// 			<KeyboardArrowDownIcon fontSize={"small"} sx={{ transform: "translateY(-2px)" }} />
						// 		</BaseNumberField.Decrement>
						// 	</InputAdornment>
						// }
						sx={{ pr: 0 }}
					/>
				)}
			/>
			<FormHelperText sx={{ ml: 0, "&:empty": { mt: 0 } }}>
				{`Enter value between ${min} and ${max}`}
			</FormHelperText>
		</BaseNumberField.Root>
	);
};
