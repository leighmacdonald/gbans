import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import PermMediaIcon from "@mui/icons-material/PermMedia";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import { useMemo } from "react";
import "video-react/dist/video-react.css";
import { type Asset, MediaTypes, mediaType } from "../../schema/asset";
import { Heading } from "../Heading";
import { ImageAsset } from "../ImageAsset";
import { VideoAsset } from "../VideoAsset";

export const AssetViewer = NiceModal.create((asset: Asset) => {
	const modal = useModal();

	const content = useMemo(() => {
		switch (mediaType(asset.mime_type)) {
			case MediaTypes.image:
				return <ImageAsset asset={asset} />;
			case MediaTypes.video:
				return <VideoAsset asset={asset} />;
			default:
				return;
		}
	}, [asset]);

	return (
		<Dialog fullWidth {...muiDialogV5(modal)} fullScreen={mediaType(asset.mime_type) === MediaTypes.image}>
			<DialogTitle component={Heading} iconLeft={<PermMediaIcon />}>
				{`${asset.name}`}
			</DialogTitle>

			<DialogContent>{content}</DialogContent>

			<DialogActions>{/*<CloseButton onClick={modal.hide} />*/}</DialogActions>
		</Dialog>
	);
});
