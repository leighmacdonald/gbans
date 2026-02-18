import Card from "@mui/material/Card";
import CardActionArea from "@mui/material/CardActionArea";
import CardMedia from "@mui/material/CardMedia";
import { assetURL } from "../api/media";
import type { Asset } from "../schema/asset";

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
