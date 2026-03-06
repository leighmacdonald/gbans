import { NumberField as BaseNumberField } from "@base-ui/react/number-field";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import InputLabel from "@mui/material/InputLabel";
import OutlinedInput from "@mui/material/OutlinedInput";
import { useStore } from "@tanstack/react-form";
import { useId, useMemo } from "react";
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

	const message = useMemo(() => {
		if (min !== undefined && max !== undefined) {
			return `Enter value between ${min} and ${max}`;
		} else if (min !== undefined) {
			return `Enter value above ${min}`;
		} else if (max !== undefined) {
			return `Enter value below ${max}`;
		} else {
			return `Enter a number`;
		}
	}, [min, max]);

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
						error={errors.length > 0}
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
			<FormHelperText sx={{ ml: 0, "&:empty": { mt: 0 } }}>{message}</FormHelperText>
		</BaseNumberField.Root>
	);
};
