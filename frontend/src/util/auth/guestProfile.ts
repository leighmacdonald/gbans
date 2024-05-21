import { defaultAvatarHash, PermissionLevel, UserProfile } from '../../api';

export const guestProfile: UserProfile = {
    steam_id: '',
    permission_level: PermissionLevel.Guest,
    avatarhash: defaultAvatarHash,
    name: '',
    ban_id: 0,
    muted: false,
    discord_id: '',
    created_on: new Date(),
    updated_on: new Date()
};
