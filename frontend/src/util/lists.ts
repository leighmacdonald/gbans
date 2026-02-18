export const noop = (): void => {};

export const sum = (list: number[]) =>
	list.reduce((acc, current) => {
		return acc + current;
	}, 0);

export const uniqCI = (values: string[]): string[] => [...new Map(values.map((s) => [s.toLowerCase(), s])).values()];
