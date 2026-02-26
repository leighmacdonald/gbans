import NiceModal, { useModal } from "@ebay/nice-modal-react";
import CloseIcon from "@mui/icons-material/Close";
import Button, { type ButtonProps } from "@mui/material/Button";
import { type ReactNode, useCallback } from "react";
import { useFormContext } from "../../../contexts/formContext.tsx";
import { ConfirmationModal } from "../../modal/ConfirmationModal.tsx";

type Props = {
	label?: string;
	labelLoading?: string;
	disabled?: boolean;
	startIcon?: ReactNode;
	endIcon?: ReactNode;
	onClick?: () => void | Promise<void>;
} & ButtonProps;

export const CloseButton = (props: Props) => {
	const form = useFormContext();
	const modal = useModal(ConfirmationModal);

	const onClick = useCallback(async () => {
		if (form.state.isDirty) {
			try {
				const confirmed = await NiceModal.show(ConfirmationModal, {
					title: `Are you sure you want to close? You have unsaved changes.`,
				});

				if (!confirmed) {
					return;
				}
			} catch {
				return;
			}
		}

		await modal.hide();
	}, [modal, form.state.isDirty]);

	return (
		<form.Subscribe selector={(state) => state.isSubmitting}>
			{() => (
				<Button {...props} onClick={props.onClick ?? onClick} type="button" startIcon={<CloseIcon />}>
					{props.label ?? "Close"}
				</Button>
			)}
		</form.Subscribe>
	);
};
