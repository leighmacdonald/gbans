import UndoIcon from "@mui/icons-material/Undo";
import Button, { type ButtonProps } from "@mui/material/Button";
import type { ReactNode } from "react";
import { useFormContext } from "../../../contexts/formContext.tsx";

type Props = {
	label?: string;
	labelLoading?: string;
	disabled?: boolean;
	startIcon?: ReactNode;
	endIcon?: ReactNode;
} & ButtonProps;

export const ResetButton = (props: Props) => {
	const form = useFormContext();

	return (
		<form.Subscribe selector={(state) => state.isSubmitting}>
			{() => (
				<Button
					{...props}
					onClick={() => {
						form.reset();
					}}
					type="reset"
					color={"warning"}
					startIcon={<UndoIcon />}
				>
					{props.label ?? "Reset"}
				</Button>
			)}
		</form.Subscribe>
	);
};
