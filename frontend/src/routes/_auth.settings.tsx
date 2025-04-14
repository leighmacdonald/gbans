import { useMemo, useState } from 'react';
import { useModal } from '@ebay/nice-modal-react';
import CableIcon from '@mui/icons-material/Cable';
import ConstructionIcon from '@mui/icons-material/Construction';
import DeleteIcon from '@mui/icons-material/Delete';
import ForumIcon from '@mui/icons-material/Forum';
import LoginIcon from '@mui/icons-material/Login';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import SettingsIcon from '@mui/icons-material/Settings';
import SettingsInputComponentIcon from '@mui/icons-material/SettingsInputComponent';
import SportsEsportsIcon from '@mui/icons-material/SportsEsports';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useLoaderData, useNavigate } from '@tanstack/react-router';
import { z } from 'zod';
import {
    apiDiscordLogout,
    apiDiscordUser,
    apiGetDiscordLogin,
    apiGetPersonSettings,
    apiSavePersonSettings,
    discordAvatarURL,
    PermissionLevel,
    PersonSettings
} from '../api';
import { apiGetPatreonLogin, apiGetPatreonLogout } from '../api/patreon.ts';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { Title } from '../component/Title.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { MarkdownField, mdEditorRef } from '../component/field/MarkdownField.tsx';
import { ModalConfirm } from '../component/modal';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
import { SubHeading, TabButton, TabSection } from './_admin.admin.settings.tsx';

const settingsSchema = z.object({
    section: z.enum(['general', 'forums', 'connections', 'game']).optional().default('general')
});

export const Route = createFileRoute('/_auth/settings')({
    component: ProfileSettings,
    validateSearch: (search) => settingsSchema.parse(search),
    loader: async ({ context }) => {
        return await context.queryClient.ensureQueryData({
            queryKey: ['settings'],
            queryFn: async () => {
                return await apiGetPersonSettings();
            }
        });
    }
});

interface SettingsValues {
    forum_signature: string;
    forum_profile_messages: boolean;
    stats_hidden: boolean;
    center_projectiles: boolean;
}

type userSettingTabs = 'general' | 'connections' | 'forums' | 'game';

function ProfileSettings() {
    const { sendFlash, sendError } = useUserFlashCtx();
    const { profile, hasPermission } = Route.useRouteContext();
    const settings = useLoaderData({ from: '/_auth/settings' }) as PersonSettings;
    const { section } = Route.useSearch();
    const [tab, setTab] = useState<userSettingTabs>(section);
    const navigate = useNavigate();
    const { appInfo } = useAppInfoCtx();

    const mutation = useMutation({
        mutationFn: async (values: SettingsValues) => {
            return await apiSavePersonSettings(
                values.forum_signature,
                values.forum_profile_messages,
                values.stats_hidden,
                values.center_projectiles ?? false
            );
        },
        onSuccess: async () => {
            mdEditorRef.current?.setMarkdown('');
            sendFlash('success', 'Updated Settings');
        },
        onError: sendError
    });

    const onTabClick = async (section: userSettingTabs) => {
        setTab(section);
        await navigate({ to: '/settings', replace: true, search: { section } });
    };

    return (
        <>
            <Title>User Settings</Title>

            <ContainerWithHeader title={'User Settings'} iconLeft={<ConstructionIcon />}>
                <Grid container spacing={2}>
                    <Grid size={{ xs: 4, sm: 3, md: 2 }} padding={0}>
                        <Stack spacing={1} padding={2}>
                            <TabButton
                                tab={'general'}
                                onClick={onTabClick}
                                icon={<SettingsIcon />}
                                currentTab={tab}
                                label={'General'}
                            />
                            <TabButton
                                tab={'game'}
                                onClick={onTabClick}
                                icon={<SportsEsportsIcon />}
                                currentTab={tab}
                                label={'Gameplay'}
                            />
                            {hasPermission(PermissionLevel.Moderator) && (
                                <TabButton
                                    tab={'forums'}
                                    onClick={onTabClick}
                                    icon={<ForumIcon />}
                                    currentTab={tab}
                                    label={'Forums'}
                                />
                            )}
                            {(appInfo.patreon_enabled || appInfo.discord_enabled) && (
                                <TabButton
                                    tab={'connections'}
                                    onClick={onTabClick}
                                    icon={<CableIcon />}
                                    currentTab={tab}
                                    label={'Connections'}
                                />
                            )}
                        </Stack>
                    </Grid>
                    <GeneralSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <GameplaySection tab={tab} settings={settings} mutate={mutation.mutate} />
                    {hasPermission(PermissionLevel.Moderator) && (
                        <ForumSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    )}
                    <ConnectionsSection
                        tab={tab}
                        settings={settings}
                        mutate={mutation.mutate}
                        patreon_id={profile.patreon_id}
                    />
                </Grid>
            </ContainerWithHeader>
        </>
    );
}

const GeneralSection = ({
    tab,
    settings,
    mutate
}: {
    tab: userSettingTabs;
    settings: PersonSettings;
    mutate: (s: PersonSettings) => void;
}) => {
    const [notifPerms, setNotifPerms] = useState(Notification.permission);

    const notificationsSupported = useMemo(() => {
        return 'Notification' in window;
    }, []);

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ...value });
        },
        defaultValues: {
            stats_hidden: settings.stats_hidden
        }
    });

    const togglePerms = async () => {
        setNotifPerms(await Notification.requestPermission());
    };

    return (
        <TabSection
            tab={'general'}
            currentTab={tab}
            label={'General'}
            description={'Core settings that dont fit into a subcategory'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    {notificationsSupported && (
                        <Grid size={{ xs: 12 }}>
                            <div>
                                <Grid container spacing={1}>
                                    <Grid size={{ xs: 12, md: 6 }}>
                                        <Stack spacing={1}>
                                            <Typography>Show desktop notifications?</Typography>
                                            {notifPerms != 'granted' ? (
                                                <Button
                                                    sx={{ width: 300 }}
                                                    variant={'contained'}
                                                    color={'success'}
                                                    onClick={togglePerms}
                                                    startIcon={<NotificationsActiveIcon />}
                                                >
                                                    Enable Desktop Notifications
                                                </Button>
                                            ) : (
                                                <Tooltip
                                                    title={
                                                        'Please see the links to the right for instructions on how to disable'
                                                    }
                                                >
                                                    <span>
                                                        <Button
                                                            sx={{ width: 300 }}
                                                            disabled={true}
                                                            variant={'contained'}
                                                            color={'success'}
                                                            onClick={togglePerms}
                                                            startIcon={<NotificationsActiveIcon />}
                                                        >
                                                            Desktop Notifications Enabled
                                                        </Button>
                                                    </span>
                                                </Tooltip>
                                            )}
                                        </Stack>
                                    </Grid>
                                    <Grid size={{ xs: 12, md: 6 }}>
                                        <Typography>How to disable notifications: </Typography>
                                        <List>
                                            <ListItemText>
                                                <Link
                                                    href={
                                                        'https://support.google.com/chrome/answer/114662?sjid=2540186662959230327-NC&visit_id=638611827496769425-656746251&rd=1'
                                                    }
                                                >
                                                    Google Chrome
                                                </Link>
                                            </ListItemText>
                                            <ListItemText>
                                                <Link
                                                    href={
                                                        'https://support.mozilla.org/en-US/kb/push-notifications-firefox'
                                                    }
                                                >
                                                    Mozilla Firefox
                                                </Link>
                                            </ListItemText>
                                        </List>
                                    </Grid>
                                </Grid>
                            </div>
                        </Grid>
                    )}
                    <Grid size={{ xs: 12 }}>
                        <Field
                            name={'stats_hidden'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={({ state, handleBlur, handleChange }) => {
                                return (
                                    <CheckboxSimple
                                        state={state}
                                        handleBlur={handleBlur}
                                        handleChange={handleChange}
                                        label={'Hide personal stats on profile'}
                                    />
                                );
                            }}
                        />
                        <SubHeading>It is still viewable by yourself.</SubHeading>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </Grid>
            </form>
        </TabSection>
    );
};

const GameplaySection = ({
    tab,
    settings,
    mutate
}: {
    tab: userSettingTabs;
    settings: PersonSettings;
    mutate: (s: PersonSettings) => void;
}) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ...value });
        },
        defaultValues: {
            center_projectiles: settings.center_projectiles ?? false
        }
    });

    return (
        <TabSection tab={'game'} currentTab={tab} label={'Gameplay'} description={'Configure your in-game clientprefs'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid size={{ xs: 12 }}>
                        <Field
                            name={'center_projectiles'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={({ state, handleBlur, handleChange }) => {
                                return (
                                    <CheckboxSimple
                                        state={state}
                                        handleBlur={handleBlur}
                                        handleChange={handleChange}
                                        label={'Use center projectiles'}
                                    />
                                );
                            }}
                        />
                        <SubHeading>Applies to all projectile weapons</SubHeading>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </Grid>
            </form>
        </TabSection>
    );
};

const ForumSection = ({
    tab,
    settings,
    mutate
}: {
    tab: userSettingTabs;
    settings: PersonSettings;
    mutate: (s: PersonSettings) => void;
}) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ...value });
        },
        defaultValues: {
            forum_signature: settings.forum_signature,
            forum_profile_messages: settings.forum_profile_messages
        }
    });

    return (
        <TabSection tab={'forums'} currentTab={tab} label={'Forums'} description={'Configure forum features'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid size={{ xs: 12 }}>
                        <Field
                            name={'forum_signature'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <MarkdownField {...props} label={'Your forum signature'} rows={10} />;
                            }}
                        />
                        <SubHeading>It is still viewable by yourself.</SubHeading>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Field
                            name={'forum_profile_messages'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={({ state, handleBlur, handleChange }) => {
                                return (
                                    <CheckboxSimple
                                        state={state}
                                        handleBlur={handleBlur}
                                        handleChange={handleChange}
                                        label={'Enable people to sign your profile.'}
                                    />
                                );
                            }}
                        />
                        <SubHeading>It is still viewable by yourself.</SubHeading>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </Grid>
            </form>
        </TabSection>
    );
};

const ConnectionsSection = ({
    tab,
    patreon_id
}: {
    tab: userSettingTabs;
    settings: PersonSettings;
    mutate: (s: PersonSettings) => void;
    patreon_id: string;
}) => {
    const queryClient = useQueryClient();
    const { profile, login } = Route.useRouteContext();
    const { sendFlash } = useUserFlashCtx();
    const confirmModal = useModal(ModalConfirm);
    const { appInfo } = useAppInfoCtx();

    const { data: user, isLoading } = useQuery({
        queryKey: ['discordProfile', { steamID: profile.steam_id }],
        queryFn: async () => {
            return apiDiscordUser();
        }
    });

    const followPatreonCallback = async () => {
        const result = await queryClient.fetchQuery({ queryKey: ['callbackPatreon'], queryFn: apiGetPatreonLogin });
        window.open(result.url, '_self');
    };

    const followDiscordCallback = async () => {
        const result = await queryClient.fetchQuery({ queryKey: ['callbackDiscord'], queryFn: apiGetDiscordLogin });
        window.open(result.url, '_self');
    };

    const onForgetDiscord = async () => {
        const confirmed = await confirmModal.show({
            title: 'Are you sure you want to remove discord connection?',
            children: 'You will need to reconnect if you want to use related features again in the future.'
        });
        if (!confirmed) {
            return;
        }
        try {
            await queryClient.fetchQuery({
                queryKey: ['discordForget', { id: user?.id }],
                queryFn: apiDiscordLogout
            });

            queryClient.setQueryData(['discordProfile', { steamID: profile.steam_id }], {});

            sendFlash('success', 'Logged out successfully');
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Could not logout fully');
        }
    };

    const onForget = async () => {
        const confirmed = await confirmModal.show({
            title: 'Are you sure you want to remove patreon connection?',
            children: 'You will need to reconnect if you want to use related features again in the future.'
        });
        if (!confirmed) {
            return;
        }
        try {
            await queryClient.fetchQuery({
                queryKey: ['patreonForget', { patreon_id }],
                queryFn: apiGetPatreonLogout
            });
            login({ ...profile, discord_id: '' });
            sendFlash('success', 'Logged out successfully');
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Could not logout fully');
        }
    };

    return (
        <TabSection
            tab={'connections'}
            currentTab={tab}
            label={'Connections'}
            description={'Configure your 3rd party connections to us.'}
        >
            <Grid container spacing={2} padding={0}>
                {appInfo.patreon_enabled ? (
                    patreon_id ? (
                        <Grid size={{ xs: 12 }}>
                            <Typography variant={'h3'}>Patreon</Typography>
                            <Box>
                                <SubHeading>
                                    You are currently authenticated to us as:{' '}
                                    <Link href={`https://www.patreon.com/user/creators?u=${patreon_id}`}>
                                        {patreon_id}
                                    </Link>
                                </SubHeading>
                            </Box>
                            <Button
                                color={'error'}
                                startIcon={<DeleteIcon />}
                                variant={'contained'}
                                onClick={onForget}
                                fullWidth={false}
                            >
                                Disconnect Patreon
                            </Button>
                        </Grid>
                    ) : (
                        <Grid size={{ xs: 12 }}>
                            <Button
                                variant={'contained'}
                                color={'success'}
                                onClick={followPatreonCallback}
                                startIcon={<SettingsInputComponentIcon />}
                            >
                                Connect Patreon
                            </Button>
                        </Grid>
                    )
                ) : (
                    <></>
                )}
                {appInfo.discord_enabled ? (
                    !isLoading && user?.username ? (
                        <Grid size={{ xs: 12 }}>
                            <Typography>You are connected to us as: {user.username}</Typography>
                            <Button
                                variant={'contained'}
                                color={'error'}
                                onClick={onForgetDiscord}
                                startIcon={<Avatar src={discordAvatarURL(user)} sx={{ height: 20, width: 20 }} />}
                            >
                                Disconnect Discord
                            </Button>
                        </Grid>
                    ) : (
                        <Grid size={{ xs: 12 }}>
                            <Button
                                variant={'contained'}
                                color={'success'}
                                onClick={followDiscordCallback}
                                startIcon={<LoginIcon />}
                            >
                                Connect Discord
                            </Button>
                        </Grid>
                    )
                ) : (
                    <></>
                )}
            </Grid>
        </TabSection>
    );
};
