import { noop } from 'lodash-es';

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
        // eslint-disable-next-line no-console
        console.log(`[${level}] ${msg}`);
    } else {
        // TODO: Log client side errors on server
        noop();
    }
};
