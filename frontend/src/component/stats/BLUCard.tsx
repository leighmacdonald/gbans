import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import blu_logo from "../../icons/blu_logo.png";
import { blu } from "../../theme";

export const BLUCard = ({ score, winner }: { score: number; winner: boolean }) => {
	return (
		<Box
			sx={{
				backgroundColor: blu,
				background: `url(${blu_logo})`,
				backgroundRepeat: "no-repeat",
				// backgroundSize: "cover",
				backgroundPosition: "right",
			}}
			flex={1}
			height={65}
			paddingLeft={2}
			paddingTop={1}
		>
			<Typography variant="h1" fontFamily={"TF2 Build"} color={winner ? "success" : "error"}>
				{score}
			</Typography>
		</Box>
	);
};
