import { noop } from './lists.ts';

export type runModeNames = 'development' | 'production';
export const runMode: runModeNames =
    (process.env.NODE_ENV as runModeNames) || 'development';

export enum Level {
    info = 0,
    warn = 1,
    err = 2
}

export const log = (msg: unknown, level: Level = Level.err): void => {
    if (runMode === 'development') {
        if (
            (msg as object).hasOwnProperty('message') &&
            (msg as Error).name != 'AbortError'
        ) {
            // eslint-disable-next-line no-console
            console.log(`[${level}] ${msg}`);
        }
    } else {
        // TODO: Log client side errors on server
        noop();
    }
};

export const logErr = (exception: unknown): void => {
    if ((exception as object).hasOwnProperty('name')) {
        if ((exception as Error).name !== 'AbortError') {
            return log(exception, Level.err);
        }
    }
    return log(exception, Level.err);
};
