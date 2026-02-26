import { useEffect, useState } from "react";

export function useSessionStorage<T>(key: string, initialValue: T) {
	const state = useState<T>(() => {
		const stored = sessionStorage.getItem(key);
		return stored ? JSON.parse(stored) : initialValue;
	});

	useEffect(() => {
		sessionStorage.setItem(key, JSON.stringify(state[0]));
	}, [key, state[0]]);

	return state;
}
