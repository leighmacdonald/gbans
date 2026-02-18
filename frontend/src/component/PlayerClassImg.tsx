import type { MouseEventHandler } from "react";
import demoClassImg from "../icons/class_demoman.png";
import engineerClassImg from "../icons/class_engineer.png";
import heavyClassImg from "../icons/class_heavy.png";
import medicClassImg from "../icons/class_medic.png";
import pyroClassImg from "../icons/class_pyro.png";
import scoutClassImg from "../icons/class_scout.png";
import sniperClassImg from "../icons/class_sniper.png";
import soldierClassImg from "../icons/class_soldier.png";
import spyClassImg from "../icons/class_spy.png";
import { PlayerClass, type PlayerClassEnum, PlayerClassNames } from "../schema/stats.ts";

export interface PlayerClassImgProps {
	cls: PlayerClassEnum;
	size?: number;
	onMouseEnter?: MouseEventHandler | undefined;
	onMouseLeave?: MouseEventHandler | undefined;
}

export const PlayerClassImg = ({ cls, size = 24, onMouseEnter, onMouseLeave }: PlayerClassImgProps) => {
	switch (cls) {
		case PlayerClass.Demo:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={demoClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Scout:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={scoutClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Engineer:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={engineerClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Pyro:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={pyroClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Heavy:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={heavyClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Sniper:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={sniperClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Spy:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={spyClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		case PlayerClass.Soldier:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={soldierClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
		default:
			return (
				<img
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					src={medicClassImg}
					alt={PlayerClassNames[cls]}
					width={size}
					height={size}
				/>
			);
	}
};
