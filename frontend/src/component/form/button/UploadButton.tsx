import CloudUploadIcon from "@mui/icons-material/CloudUpload";
import Button from "@mui/material/Button";
import { useCallback } from "react";
import { apiSaveAsset } from "../../../api/media";
import type { Asset } from "../../../schema/asset";
import { VisuallyHiddenInput } from "../field/VisuallyHiddenInput";

type UploadButtonProps = {
	label?: string;
	onSuccess?: (file: Asset) => void;
	icon?: React.ReactNode;
};

export const UploadButton = ({ onSuccess, icon, label = "Upload files" }: UploadButtonProps) => {
	const onChange = useCallback(
		async (e: React.ChangeEvent<HTMLInputElement>) => {
			if (e.target.files) {
				const asset = await apiSaveAsset(e.target.files[0]);
				if (onSuccess) {
					onSuccess(asset);
				}
			}
		},
		[onSuccess],
	);

	return (
		<Button
			component="label"
			role={undefined}
			variant="contained"
			tabIndex={-1}
			startIcon={icon ?? <CloudUploadIcon />}
		>
			{label}
			<VisuallyHiddenInput type="file" onChange={onChange} multiple accept=".png" />
		</Button>
	);
};
