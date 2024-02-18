import { defaultAvatarHash, PermissionLevel, UserProfile } from '../api';

export const GuestProfile: UserProfile = {
    updated_on: new Date(),
    created_on: new Date(),
    permission_level: PermissionLevel.Guest,
    discord_id: '',
    avatarhash: defaultAvatarHash,
    steam_id: '',
    ban_id: 0,
    name: 'Guest',
    muted: false
};
