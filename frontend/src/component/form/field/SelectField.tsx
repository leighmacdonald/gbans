import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import InputLabel from "@mui/material/InputLabel";
import Select, { type SelectProps } from "@mui/material/Select";
import { useStore } from "@tanstack/react-form";
import type { ReactNode } from "react";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import { renderHelpText } from "./renderHelpText.ts";

type Props<TData> = {
	label?: string;
	labelLoading?: string;
	items: TData[];
	renderItem: (item: TData) => ReactNode;
	helperText?: ReactNode | string;
	handleChange?: (item: TData) => void;
} & SelectProps;

export const SelectField = <TData,>(props: Props<TData>) => {
	const field = useFieldContext<TData>();
	const errors = useStore(field.store, (state) => state.meta.errors);

	return (
		<FormControl fullWidth error={errors.length > 0}>
			<InputLabel id={`select-label-${props.name}`}>{props.label}</InputLabel>
			<Select
				disabled={props.disabled}
				color={field.state.meta.isValid ? "success" : props.color}
				onClick={props.onClick}
				value={field.state.value}
				multiple={props.multiple}
				id={`select-${props.name}`}
				fullWidth
				onChange={(event) => {
					field.handleChange(event.target.value as TData);
					if (props.handleChange) {
						props.handleChange(event.target.value as TData);
					}
				}}
			>
				{props.items.map((i) => {
					return props.renderItem(i);
				})}
			</Select>
			<FormHelperText>{renderHelpText(errors, props.helperText)}</FormHelperText>
		</FormControl>
	);
};

export default SelectField;
