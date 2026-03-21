type colourVariant = "dark" | "light";

const colourMap: Record<colourVariant, Record<string, string>> = {
	light: {},
	dark: {},
};

export const stringToColour = (str: string) => {
	const variant = (localStorage.getItem("theme") ?? "light") as colourVariant;

	if (str in colourMap[variant]) {
		return colourMap[variant][str];
	}
	const colourH = Math.floor(Math.random() * 359);
	const colourS = Math.floor(Math.random() * 40) + 60;
	const colourL = Math.floor(Math.random() * 15) + (variant === "light" ? 20 : 80);
	const randColour = `hsl(${colourH}, ${colourS}%, ${colourL}%)`;

	colourMap[variant][str] = randColour;

	return randColour;
};
