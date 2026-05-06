import { Box } from "@mui/material";
import ReactPlayer from "react-player";
import type { Asset } from "../rpc/asset/v1/asset_pb.ts";
import { assetURL } from "../util/strings.ts";

export const VideoAsset = ({ asset }: { asset: Asset }) => (
	<Box>
		<ReactPlayer src={assetURL(asset)} />
	</Box>
);
