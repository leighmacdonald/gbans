import { useEffect, useRef } from "react";
import SteamID from "steamid";
import backpacktfIcon from "../icons/sites/backpacktf.png";
import demosIcon from "../icons/sites/demos.svg";
import logsIcon from "../icons/sites/logs.png";
import moretfIcon from "../icons/sites/moretf.png";
import ozfortressIcon from "../icons/sites/ozfortress.png";
import rglIcon from "../icons/sites/rgl.png";
import steamIcon from "../icons/sites/steam.png";
import trendsIcon from "../icons/sites/trends.png";

export const to = (url: string) => {
	window.open(url, "_self");
};

export interface LinkProps {
	title: string;
	url: string;
	// Bundled site icon. Absent when the site blocks automated fetching.
	icon?: string;
}

export const useScrollToLocation = () => {
	const scrolledRef = useRef(false);
	// const { hash } = useLocation();
	// TODO fix
	const hash = "#";
	const hashRef = useRef(hash);

	useEffect(() => {
		if (hash) {
			// We want to reset if the hash has changed
			if (hashRef.current !== hash) {
				hashRef.current = hash;
				scrolledRef.current = false;
			}

			// only attempt to scroll if we haven't yet (this could have just reset above if hash changed)
			if (!scrolledRef.current) {
				const id = hash.replace("#", "");
				const element = document.getElementById(id);
				if (element) {
					element.scrollIntoView({ behavior: "smooth" });
					scrolledRef.current = true;
				}
			}
		}
	});
};

export const createExternalLinks = (steamIdStr: string): LinkProps[] => {
	const steamId = new SteamID(steamIdStr);

	return [
		{
			title: "Steam",
			url: `https://steamcommunity.com/profiles/${steamId.getSteamID64()}`,
			icon: steamIcon,
		},
		{
			title: "RGL",
			url: `https://rgl.gg/Public/PlayerProfile.aspx?p=${steamId.getSteamID64()}`,
			icon: rglIcon,
		},
		{
			title: "OzFortress",
			url: `https://ozfortress.com/users?q=${steamId.getSteamID64()}`,
			icon: ozfortressIcon,
		},
		{
			title: "logs.tf",
			url: `https://logs.tf/profile/${steamId.getSteamID64()}`,
			icon: logsIcon,
		},
		{
			title: "more.tf",
			url: `https://more.tf/profile/${steamId.getSteamID64()}`,
			icon: moretfIcon,
		},
		{
			title: "demos.tf",
			url: `https://demos.tf/profiles/${steamId.getSteamID64()}`,
			icon: demosIcon,
		},
		{
			title: "backpack.tf",
			url: `https://backpack.tf/profiles/${steamId.getSteamID64()}`,
			icon: backpacktfIcon,
		},
		{
			title: "trends.tf",
			url: `https://trends.tf/player/${steamId.getSteamID64()}`,
			icon: trendsIcon,
		},
		{
			title: "steamid.uk",
			url: `https://steamid.uk/profile/${steamId.getSteamID64()}`,
		},
	];
};
