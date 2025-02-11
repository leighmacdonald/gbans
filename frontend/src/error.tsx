import { emptyOrNullString } from './util/types.ts';

export enum ErrorCode {
    InvalidMimetype,
    DependencyMissing,
    PermissionDenied,
    Unknown,
    LoginRequired,
    NotFound = 5
}

export class ApiError extends Error {
    public code: ErrorCode;

    constructor(code: ErrorCode, message: string, options?: never) {
        if (emptyOrNullString(message)) {
            switch (code) {
                case ErrorCode.InvalidMimetype:
                    message = 'Forbidden file format (mimetype)';
                    break;
                case ErrorCode.DependencyMissing:
                    message = 'Dependency missing, cannot continue';
                    break;
                case ErrorCode.PermissionDenied:
                    message = 'Permission Denied';
                    break;
                case ErrorCode.LoginRequired:
                    message = 'Please Login';
                    break;
                case ErrorCode.NotFound:
                    message = 'Not Found';
                    break;
                default:
                    message = '🤯 🤯 🤯 Something went wrong 🤯 🤯 🤯';
            }
        }

        super(options);
        this.code = code;
        this.message = message;
    }
}
