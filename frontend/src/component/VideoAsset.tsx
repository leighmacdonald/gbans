import { Box } from "@mui/material";
import { Player } from "video-react";
import { assetURL } from "../api/media";
import type { Asset } from "../schema/asset";

export const VideoAsset = ({ asset }: { asset: Asset }) => (
	<Box>
		<Player>
			<source src={assetURL(asset)} type={asset.mime_type} />
		</Player>
	</Box>
);
