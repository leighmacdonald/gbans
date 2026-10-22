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

/**
 * Form close button that prompts for confirmation when the form has unsaved changes.
 *
 * When `onClick` is supplied, the caller owns the close action (eg. exiting an
 * inline edit mode or hiding its own modal) and `useModal()` is never called,
 * keeping the button safe outside a NiceModal context.
 */
export const CloseButton = (props: Props) => {
	if (props.onClick) {
		return <PlainCloseButton {...props} />;
	}

	return <ModalCloseButton {...props} />;
};

const PlainCloseButton = (props: Props) => {
	const form = useFormContext();
	const { label, ...rest } = props;

	return (
		<form.Subscribe selector={(state) => state.isSubmitting}>
			{() => (
				<Button {...rest} type="button" startIcon={<CloseIcon />}>
					{label ?? "Close"}
				</Button>
			)}
		</form.Subscribe>
	);
};

const ModalCloseButton = (props: Omit<Props, "onClick">) => {
	const form = useFormContext();
	// Must be the current modal's handle: useModal(ConfirmationModal) would
	// hide the (unopened) confirmation dialog instead of this modal.
	const modal = useModal();

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

	const { label, ...rest } = props;

	return (
		<form.Subscribe selector={(state) => state.isSubmitting}>
			{() => (
				<Button {...rest} onClick={onClick} type="button" startIcon={<CloseIcon />}>
					{label ?? "Close"}
				</Button>
			)}
		</form.Subscribe>
	);
};

export default CloseButton;
