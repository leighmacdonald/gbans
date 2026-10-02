// Runs `run` at most once per key while its promise is pending, returning
// the shared promise to every concurrent caller. Once the promise settles
// (success or failure) the key is released, so a later caller starts a fresh
// run. This lets effects that may be invoked more than once for the same
// logical operation (React StrictMode double-mounts in development) share a
// single side effect while each invocation keeps its own cleanup state.
const inFlight = new Map<string, Promise<void>>();

export const once = (key: string, run: () => Promise<void>): Promise<void> => {
	const existing = inFlight.get(key);
	if (existing) {
		return existing;
	}

	const promise = (async () => {
		try {
			await run();
		} finally {
			inFlight.delete(key);
		}
	})();

	inFlight.set(key, promise);

	return promise;
};
