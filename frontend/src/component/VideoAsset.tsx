import { Box } from "@mui/material";
import ReactPlayer from "react-player";
import { assetURL } from "../api/media";
import type { Asset } from "../schema/asset";

export const VideoAsset = ({ asset }: { asset: Asset }) => (
	<Box>
		<ReactPlayer src={assetURL(asset)} />
	</Box>
);
