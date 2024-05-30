import loadable from '@loadable/component';
import { APIError, ErrorCode } from '../api';

const LoginPage = loadable(() => import('../page/LoginPage'));
const PageNotFoundPage = loadable(() => import('../page/PageNotFoundPage'));

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