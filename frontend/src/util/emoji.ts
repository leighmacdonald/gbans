export const getFlagEmoji = (countryCode: string) => {
	const codePoints = countryCode
		.toUpperCase()
		.split("")
		.map((char: string) => 127397 + char.charCodeAt(0));
	return String.fromCodePoint(...codePoints);
};
