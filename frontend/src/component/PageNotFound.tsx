import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";

export function PageNotFound() {
	const heading = "Not Found";
	// const error = null;

	return (
		<Grid container size={{ xs: 12 }} padding={2}>
			<Grid size={{ xs: 12 }} alignContent={"center"}>
				<Typography align={"center"} variant={"h1"}>
					{heading}
				</Typography>
				{/*{error && (*/}
				{/*    <Typography align={'center'} variant={'subtitle1'}>*/}
				{/*        {error}*/}
				{/*    </Typography>*/}
				{/*)}*/}
			</Grid>
		</Grid>
	);
}
