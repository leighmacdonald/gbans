import Card from "@mui/material/Card";
import CardActionArea from "@mui/material/CardActionArea";
import CardMedia from "@mui/material/CardMedia";
import type { Asset } from "../rpc/asset/v1/asset_pb.ts";
import { assetURL } from "../util/strings.ts";

export const ImageAsset = ({ asset }: { asset: Asset }) => {
	return (
		<Card>
			<CardActionArea>
				<CardMedia
					component="img"
					//height="140"
					image={assetURL(asset)}
					alt={asset.name}
				/>
			</CardActionArea>
		</Card>
	);
};
