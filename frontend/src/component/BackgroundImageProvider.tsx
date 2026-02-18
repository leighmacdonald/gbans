import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import RefreshIcon from "@mui/icons-material/Refresh";
import { GlobalStyles, Stack } from "@mui/material";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useEffect, useState } from "react";

function getImageUrl(name: string | number) {
	// note that this does not include files in subdirectories
	if (import.meta.env.MODE === "development") {
		return `/public/bg/${name}.png`;
	} else {
		return `/bg/${name}.png`;
	}
}

const randBGNum = () => {
	const numberOfBgImages = 265;

	return Math.floor(Math.random() * numberOfBgImages);
};

const getPausedState = () => {
	const bgPaused = localStorage.getItem("bg_paused");
	if (!bgPaused) {
		return false;
	}

	return bgPaused === "true";
};

const savePausedState = (state: boolean) => {
	localStorage.setItem("bg_paused", state.toString());
};

const saveBGNum = (num: number) => {
	localStorage.setItem("bg_num", num.toString());
};

const getBGNum = () => {
	return Number(localStorage.getItem("bg_num"));
};

export const BackgroundImageProvider = () => {
	const [paused, setPaused] = useState(getPausedState());
	const [num, setNum] = useState(() => {
		if (paused) {
			return getBGNum();
		}
		return randBGNum();
	});

	const setBG = (num: number) => {
		setNum(num);
		saveBGNum(num);
	};

	const togglePaused = () => {
		setPaused((prev) => {
			savePausedState(!prev);
			return !prev;
		});
	};

	useEffect(() => {
		saveBGNum(num);
	}, [num]);

	return (
		<>
			<div
				style={{
					position: "absolute",
					top: 50,
					right: 0,
				}}
			>
				<Stack direction={"row"} spacing={1}>
					<Tooltip title={"Randomly select a background image"}>
						<IconButton
							onClick={() => {
								setBG(randBGNum());
							}}
						>
							<RefreshIcon />
						</IconButton>
					</Tooltip>
					<Tooltip title={"Pause randomly selecting new backgrounds"}>
						<IconButton
							onClick={() => {
								togglePaused();
							}}
						>
							{paused ? <PlayArrowIcon /> : <PauseIcon />}
						</IconButton>
					</Tooltip>
				</Stack>
			</div>
			<GlobalStyles
				styles={{
					body: {
						backgroundColor: "#fff",
						backgroundImage: `url(${getImageUrl(num)})`,
						backgroundRepeat: "repeat",
					},
				}}
			/>
		</>
	);
};
