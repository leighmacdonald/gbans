import { APIError, ErrorCode } from '../api';
import { LoginPage } from '../page/LoginPage';
import { PageNotFoundPage } from '../page/PageNotFoundPage';

export const PermissionDenied = ({ error }: { error: APIError }) => {
    if (error.code == ErrorCode.LoginRequired) {
        return <LoginPage />;
    } else {
        return (
            <PageNotFoundPage
                heading={'Permission Denied'}
                error={error.message}
            />
        );
    }
};
