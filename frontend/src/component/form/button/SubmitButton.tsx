import CheckIcon from "@mui/icons-material/Check";
import Button, { type ButtonProps } from "@mui/material/Button";
import { useSelector } from "@tanstack/react-form";
import { useFormContext } from "../../../contexts/formContext.tsx";

type Props = {
	label?: string;
	labelLoading?: string;
} & ButtonProps;

export const SubmitButton = (props: Props) => {
	const form = useFormContext();
	const isSubmitting = useSelector(form.store, (state) => state.isSubmitting);
	const canSubmit = useSelector(form.store, (state) => state.canSubmit);

	return (
		<Button
			{...props}
			type="submit"
			startIcon={<CheckIcon />}
			loading={isSubmitting}
			disabled={props.disabled || !canSubmit}
			color={"success"}
		>
			{props.label ?? (isSubmitting ? (props.labelLoading ?? "Submitting") : "Submit")}
		</Button>
	);
};
export default SubmitButton;
