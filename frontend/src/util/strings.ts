const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

export const randomStringAlphaNum = (length: number) => {
	let result = "";

	for (let i = 0; i < length; i++) {
		result += characters.charAt(Math.floor(Math.random() * characters.length));
	}

	return result;
};

export const cidrHostCount = (cidr: string): number => {
	if (!cidr.includes("/")) {
		return 0;
	}
	const mask = parseInt(cidr.split("/")[1], 10);
	return mask === 32 ? 1 : mask === 31 ? 2 : 2 ** (32 - mask) - 2;
};
