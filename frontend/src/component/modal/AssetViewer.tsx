import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import PermMediaIcon from "@mui/icons-material/PermMedia";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import { useMemo } from "react";
import type { Asset } from "../../rpc/asset/v1/asset_pb.ts";
import { Heading } from "../Heading";
import { ImageAsset } from "../ImageAsset";
import { VideoAsset } from "../VideoAsset";

export const AssetViewer = NiceModal.create((asset: Asset) => {
	const modal = useModal();

	const content = useMemo(() => {
		switch (asset.mimeType) {
			case "image":
				return <ImageAsset asset={asset} />;
			case "video":
				return <VideoAsset asset={asset} />;
			default:
				return;
		}
	}, [asset]);

	return (
		<Dialog fullWidth {...muiDialogV5(modal)} fullScreen={asset.mimeType === "image"}>
			<DialogTitle component={Heading} iconLeft={<PermMediaIcon />}>
				{`${asset.name}`}
			</DialogTitle>

			<DialogContent>{content}</DialogContent>

			<DialogActions>{/*<CloseButton onClick={modal.hide} />*/}</DialogActions>
		</Dialog>
	);
});
