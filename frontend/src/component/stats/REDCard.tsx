import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import red_logo from "../../icons/red_logo.png";
import { red } from "../../theme";
export const REDCard = ({ score, winner }: { score: number; winner: boolean }) => {
	return (
		<Box
			sx={{
				backgroundColor: red,
				background: `url(${red_logo})`,
				backgroundRepeat: "no-repeat",
				// backgroundSize: "cover",
				backgroundPosition: "left",
			}}
			flex={1}
			height={65}
			textAlign={"right"}
			paddingRight={2}
			paddingTop={1}
		>
			{/*<BoxImg src={red_logo} />*/}
			<Typography variant="h1" fontFamily={"TF2 Build"} color={winner ? "success" : "error"}>
				{score}
			</Typography>
		</Box>
	);
};
