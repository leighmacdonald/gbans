import { useEffect, useState } from "react";

export enum StorageType {
	Local = 0,
	Session = 1,
}

export function useStorage<T>(key: string, initialValue: T, type = StorageType.Session) {
	const storeInterface = type === StorageType.Session ? sessionStorage : localStorage;
	const [state, setState] = useState<T>(() => {
		try {
			const stored = storeInterface.getItem(key);
			return stored ? JSON.parse(stored) : initialValue;
		} catch {
			return undefined;
		}
	});

	useEffect(() => {
		if (!state) {
			return;
		}
		storeInterface.setItem(key, JSON.stringify(state));
	}, [key, state, storeInterface]);

	const deleteValue = () => {
		storeInterface.removeItem(key);
	};

	return { value: state, setValue: setState, deleteValue };
}
