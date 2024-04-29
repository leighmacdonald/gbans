import { queryOptions } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useRouteContext } from '@tanstack/react-router';
import { apiGetProfile, PermissionLevel, PlayerProfile } from '../api';
import { ProfileDetails } from '../component/ProfileDetails.tsx';

export const Route = createFileRoute('/_guest/profile/$steamId')({
    component: ProfilePage,
    loader: async ({ context, abortController }) => {
        const getOwnProfile = queryOptions({
            queryKey: ['ownProfile'],
            queryFn: async () => await apiGetProfile(context.auth.userSteamID, abortController)
        });

        return context.queryClient.fetchQuery(getOwnProfile);
    }
});

function ProfilePage() {
    const { hasPermission } = useRouteContext({ from: '/_guest/profile/$steamId' });
    const loaderData = useLoaderData({ from: '/_guest/profile/$steamId' }) as PlayerProfile;

    console.log(loaderData);
    return <ProfileDetails profile={loaderData} loggedIn={hasPermission(PermissionLevel.User)} />;
}
