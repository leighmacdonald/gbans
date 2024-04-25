import loadable from '@loadable/component';
import { APIError, ErrorCode } from '../api';

const LoginPage = loadable(() => import('../routes/login.lazy.tsx'));
const PageNotFoundPage = loadable(
    () => import('../routes/pageNotFound.lazy.tsx')
);

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
