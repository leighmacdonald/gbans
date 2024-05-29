import { PropsWithChildren, ReactNode, useState } from 'react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import BugReportIcon from '@mui/icons-material/BugReport';
import DeveloperBoardIcon from '@mui/icons-material/DeveloperBoard';
import EmergencyRecordingIcon from '@mui/icons-material/EmergencyRecording';
import GradingIcon from '@mui/icons-material/Grading';
import HeadsetMicIcon from '@mui/icons-material/HeadsetMic';
import LanIcon from '@mui/icons-material/Lan';
import PaymentIcon from '@mui/icons-material/Payment';
import SettingsIcon from '@mui/icons-material/Settings';
import ShareIcon from '@mui/icons-material/Share';
import TravelExploreIcon from '@mui/icons-material/TravelExplore';
import TroubleshootIcon from '@mui/icons-material/Troubleshoot';
import WebAssetIcon from '@mui/icons-material/WebAsset';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetSettings, apiSaveSettings, Config } from '../api/admin.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { numberStringValidator } from '../util/validator/numberStringValidator.ts';

const settingsSchema = z.object({
    section: z
        .enum([
            'general',
            'filters',
            'demo',
            'patreon',
            'discord',
            'logging',
            'sentry',
            'geo_location',
            'debug',
            'local_store',
            'ssh',
            'exports'
        ])
        .optional()
        .default('general')
});

export const Route = createFileRoute('/_admin/admin/settings')({
    component: AdminServers,
    validateSearch: (search) => settingsSchema.parse(search),
    loader: ({ context }) => {
        return context.queryClient.fetchQuery({
            queryKey: ['settings'],
            queryFn: async () => {
                return await apiGetSettings();
            }
        });
    }
});

type tabs =
    | 'general'
    | 'filters'
    | 'demo'
    | 'patreon'
    | 'discord'
    | 'logging'
    | 'sentry'
    | 'geo_location'
    | 'debug'
    | 'local_store'
    | 'ssh'
    | 'exports';

type TabButtonProps = {
    label: string;
    tab: tabs;
    onClick: (tab: tabs) => void;
    currentTab: tabs;
    icon: ReactNode;
};

const TabButton = ({ currentTab, tab, label, onClick, icon }: TabButtonProps) => {
    return (
        <Button
            color={currentTab == tab ? 'secondary' : 'primary'}
            onClick={() => onClick(tab)}
            variant={'contained'}
            startIcon={icon}
            fullWidth
            title={label}
            style={{ justifyContent: 'flex-start' }}
        >
            {label}
        </Button>
    );
};

const TabSection = ({
    children,
    tab,
    currentTab,
    label,
    description
}: PropsWithChildren & { tab: tabs; currentTab: tabs; label: string; description: string }) => {
    return (
        <Grid xs={8} sm={9} md={10} display={tab == currentTab ? undefined : 'none'} marginTop={1}>
            <Typography variant={'h1'}>{label}</Typography>
            <Typography variant={'subtitle1'} marginBottom={2}>
                {description}
            </Typography>
            {children}
        </Grid>
    );
};

const SubHeading = ({ children }: PropsWithChildren) => <Typography variant={'caption'}>{children}</Typography>;

function AdminServers() {
    const { sendFlash } = useUserFlashCtx();
    const settings = Route.useLoaderData();
    const { section } = Route.useSearch();
    const navigate = useNavigate();
    const [tab, setTab] = useState<tabs>(section);

    const mutation = useMutation({
        mutationKey: ['adminSettings'],
        mutationFn: async (variables: Config) => {
            await apiSaveSettings(variables);
        },
        onSuccess: () => {
            sendFlash('success', 'Settings saved successfully');
        },
        onError: (error) => {
            sendFlash('error', `Error saving settings: ${error}`);
        }
    });

    const onTabClick = async (section: tabs) => {
        setTab(section);
        await navigate({ to: '/admin/settings', replace: true, search: { section } });
    };

    return (
        <>
            <Title>Edit Settings</Title>

            <ContainerWithHeaderAndButtons title={'System Settings'} iconLeft={<DeveloperBoardIcon />}>
                <Grid container spacing={2}>
                    <Grid xs={4} sm={3} md={2} padding={0}>
                        <Stack spacing={1} padding={2}>
                            <TabButton
                                tab={'general'}
                                onClick={onTabClick}
                                icon={<SettingsIcon />}
                                currentTab={tab}
                                label={'General'}
                            />
                            <TabButton
                                tab={'filters'}
                                onClick={onTabClick}
                                icon={<AddModeratorIcon />}
                                currentTab={tab}
                                label={'Word Filters'}
                            />
                            <TabButton
                                tab={'demo'}
                                onClick={onTabClick}
                                icon={<EmergencyRecordingIcon />}
                                currentTab={tab}
                                label={'Demos'}
                            />
                            <TabButton
                                tab={'discord'}
                                onClick={onTabClick}
                                icon={<HeadsetMicIcon />}
                                currentTab={tab}
                                label={'Discord'}
                            />
                            <TabButton
                                tab={'logging'}
                                onClick={onTabClick}
                                icon={<GradingIcon />}
                                currentTab={tab}
                                label={'Logging'}
                            />
                            <TabButton
                                tab={'sentry'}
                                onClick={onTabClick}
                                icon={<TroubleshootIcon />}
                                currentTab={tab}
                                label={'Sentry'}
                            />
                            <TabButton
                                tab={'geo_location'}
                                onClick={onTabClick}
                                icon={<TravelExploreIcon />}
                                currentTab={tab}
                                label={'Geo Database'}
                            />
                            <TabButton
                                tab={'debug'}
                                onClick={onTabClick}
                                icon={<BugReportIcon />}
                                currentTab={tab}
                                label={'Debug'}
                            />
                            <TabButton
                                tab={'local_store'}
                                onClick={onTabClick}
                                icon={<WebAssetIcon />}
                                currentTab={tab}
                                label={'Assets'}
                            />
                            <TabButton
                                tab={'ssh'}
                                onClick={onTabClick}
                                icon={<LanIcon />}
                                currentTab={tab}
                                label={'SSH'}
                            />
                            <TabButton
                                tab={'patreon'}
                                onClick={onTabClick}
                                icon={<PaymentIcon />}
                                currentTab={tab}
                                label={'Patreon'}
                            />
                            <TabButton
                                tab={'exports'}
                                onClick={onTabClick}
                                icon={<ShareIcon />}
                                currentTab={tab}
                                label={'Exports'}
                            />

                            <Typography padding={1}>
                                Note that many settings will not take effect until app restart.
                            </Typography>
                        </Stack>
                    </Grid>
                    <GeneralSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <FiltersSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <DemosSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <PatreonSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <DiscordSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <LoggingSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <SentrySection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <GeoLocationSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <LocalStoreSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <SSHSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <ExportsSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <DebugSection tab={tab} settings={settings} mutate={mutation.mutate} />
                </Grid>
            </ContainerWithHeaderAndButtons>
        </>
    );
}

const GeneralSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, general: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            srcds_log_addr: settings.general.srcds_log_addr,
            file_serve_mode: settings.general.file_serve_mode,
            steam_key: settings.general.steam_key,
            mode: settings.general.mode,
            site_name: settings.general.site_name,
            asset_url: settings.general.asset_url
        }
    });

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
                    <Grid xs={12}>
                        <Field
                            name={'site_name'}
                            validators={{
                                onChange: z.string().min(1).max(32)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Global Site Name'} />;
                            }}
                        />
                        <SubHeading>
                            This name is displayed in various places throughout the app such as the title bar and site
                            heading. It should be short and simple.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'steam_key'}
                            validators={{
                                onChange: z.string().refine(
                                    (arg) => {
                                        return arg.length == 0 || arg.length == 32;
                                    },
                                    {
                                        message: 'Steam API key must be 32 characters'
                                    }
                                )
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Steam API Key'} />;
                            }}
                        />
                        <SubHeading>
                            You can create or retrieve your API key{' '}
                            <Link href={'https://steamcommunity.com/dev/apikey'}>here</Link>
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'asset_url'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'URL path pointing to assets'} />;
                            }}
                        />
                        <SubHeading>If you have a asset under a different subdir you should change this.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'srcds_log_addr'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'UDP Log Listen Address'} />;
                            }}
                        />
                        <SubHeading>
                            What address to listen for UDP log events. host:port format. If host is empty, it will
                            listen on all available hosts.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const FiltersSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, filters: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            enabled: settings.filters.enabled,
            warning_timeout: settings.filters.warning_timeout,
            warning_limit: settings.filters.warning_limit,
            dry: settings.filters.dry,
            ping_discord: settings.filters.ping_discord,
            max_weight: settings.filters.max_weight,
            check_timeout: settings.filters.check_timeout,
            match_timeout: settings.filters.match_timeout
        }
    });

    return (
        <TabSection
            tab={'filters'}
            currentTab={tab}
            label={'Word Filters'}
            description={
                'Word filters are a form of auto-moderation that scans ' +
                'incoming chat logs and user names for matching values and handles them accordingly'
            }
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable Word Filters'} />;
                            }}
                        />
                        <SubHeading>Enable/disable the feature</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'warning_timeout'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(1, 1000000))
                            }}
                            children={(props) => {
                                return (
                                    <TextFieldSimple {...props} label={'How long until a warning expires (seconds)'} />
                                );
                            }}
                        />
                        <SubHeading>If a user gets a warning, it will expire after this duration of time.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'warning_limit'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 1000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Maximum number of warnings allowed'} />;
                            }}
                        />
                        <SubHeading>
                            A hard limit to the number of warnings a user can receive before action is taken.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'dry'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable dry run mode'} />;
                            }}
                        />
                        <SubHeading>Run the chat filters, but do not actually punish users.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'ping_discord'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Send discord notices on match'} />;
                            }}
                        />
                        <SubHeading>If discord is enabled, send filter match notices to the log channel.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'max_weight'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(1, 1000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Max Weight'} />;
                            }}
                        />
                        <SubHeading>
                            When the sum of warning weights issued to a user is greater than this value, take action
                            against the user.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'check_timeout'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(5, 300))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Check Frequency (seconds)'} />;
                            }}
                        />
                        <SubHeading>How frequent warnings will be checked for users exceeding limits.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'match_timeout'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(1, 10000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Match Timeout'} />;
                            }}
                        />
                        <SubHeading>How long it takes for a users warning to expire after being matched.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const DemosSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, demo: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            demo_cleanup_enabled: settings.demo.demo_cleanup_enabled,
            demo_cleanup_strategy: settings.demo.demo_cleanup_strategy,
            demo_cleanup_min_pct: settings.demo.demo_cleanup_min_pct,
            demo_cleanup_mount: settings.demo.demo_cleanup_mount,
            demo_count_limit: settings.demo.demo_count_limit
        }
    });

    return (
        <TabSection
            tab={'demo'}
            currentTab={tab}
            label={'Demos/SourceTV'}
            description={'How to handle demo storage and cleanup'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'demo_cleanup_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable Demo Cleanup'} />;
                            }}
                        />
                        <SubHeading>
                            Enable automatic deletion of demos. This ignores demos that have been marked as archived.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'demo_cleanup_strategy'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return (
                                    <SelectFieldSimple
                                        {...props}
                                        label={'Cleanup Strategy'}
                                        items={['pctfree', 'count']}
                                        renderMenu={(item) => {
                                            return (
                                                <MenuItem key={item} value={item}>
                                                    {item}
                                                </MenuItem>
                                            );
                                        }}
                                    />
                                );
                            }}
                        />
                        <SubHeading>Method used to determine what demos to delete.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'demo_cleanup_min_pct'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 100))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Minimum percent free space to retain'} />;
                            }}
                        />
                        <SubHeading>
                            When using the percent free strategy, defined how much free space should be retained on the
                            demo mount/volume.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'demo_cleanup_mount'}
                            validators={{
                                onChange: z.string().startsWith('/')
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Mount point to check for free space'} />;
                            }}
                        />
                        <SubHeading>The mount point that demos are stored. Used to determine free space.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'demo_count_limit'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 100000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Max amount of demos to keep'} />;
                            }}
                        />
                        <SubHeading>
                            When using the count deletion strategy, this is the maximum number of demos to keep.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const PatreonSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, patreon: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            enabled: settings.patreon.enabled,
            client_id: settings.patreon.client_id,
            client_secret: settings.patreon.client_secret,
            creator_access_token: settings.patreon.creator_access_token,
            creator_refresh_token: settings.patreon.creator_refresh_token
        }
    });

    return (
        <TabSection tab={'patreon'} currentTab={tab} label={'Patreon'} description={'Connect to patreon API'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable Patreon Integration'} />;
                            }}
                        />
                        <SubHeading>Enabled/Disable patreon integrations</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'client_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Client ID'} />;
                            }}
                        />
                        <SubHeading>Your patron client ID</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'client_secret'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 100))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Client Secret'} />;
                            }}
                        />
                        <SubHeading>Patreon app client secret</SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'creator_access_token'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Access Token'} />;
                            }}
                        />
                        <SubHeading>Access token</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'creator_refresh_token'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Refresh Token'} />;
                            }}
                        />
                        <SubHeading>Refresh token</SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const DiscordSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, discord: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            enabled: settings.discord.enabled,
            app_id: settings.discord.app_id,
            app_secret: settings.discord.app_secret,
            link_id: settings.discord.link_id,
            token: settings.discord.token,
            guild_id: settings.discord.guild_id,
            log_channel_id: settings.discord.log_channel_id,
            public_log_channel_enable: settings.discord.public_log_channel_enable,
            public_log_channel_id: settings.discord.public_log_channel_id,
            public_match_log_channel_id: settings.discord.public_match_log_channel_id,
            mod_ping_role_id: settings.discord.mod_ping_role_id,
            unregister_on_start: settings.discord.unregister_on_start
        }
    });

    return (
        <TabSection tab={'discord'} currentTab={tab} label={'Discord'} description={'Support for discord bot'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable discord bot integration'} />;
                            }}
                        />
                        <SubHeading>
                            Enabled the discord bot integration. This is self-hosted within the app. You must can create
                            a discord application{' '}
                            <Link href={'https://discord.com/developers/applications?new_application=true'}>here</Link>.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'app_id'}
                            validators={{
                                onChange: z.string().refine((arg) => arg.length == 0 || arg.length == 18)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord app ID'} />;
                            }}
                        />
                        <SubHeading>Your discord application ID.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'app_secret'}
                            validators={{
                                onChange: z.string().refine((arg) => arg.length == 0 || arg.length == 32)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord bot app secret'} />;
                            }}
                        />
                        <SubHeading>Your discord app secret.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'link_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Invite link ID'} />;
                            }}
                        />
                        <SubHeading>
                            The unique ID for your permanent discord link. This is only the unique string at the end if
                            a invite url: https://discord.gg/&lt;XXXXXXXXX&gt;, not the entire url.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'token'}
                            validators={{
                                onChange: z.string().refine((arg) => {
                                    if (arg.length == 0) {
                                        return true;
                                    }

                                    return z
                                        .string()
                                        .regex(/^\S{24}\.\S{6}\.\S{4}-\S{33}$/)
                                        .safeParse(arg).success;
                                })
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord Bot Token'} />;
                            }}
                        />
                        <SubHeading>Bot authentication token.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'guild_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord guild ID'} />;
                            }}
                        />
                        <SubHeading>
                            This is the guild id of your discord server. With discoed developer mode enabled,
                            right-click on the server title and select "Copy ID" to get the guild ID.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Log channel ID'} />;
                            }}
                        />
                        <SubHeading>Private channel id used for moderation related log events to be sent.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'public_log_channel_enable'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable public log channel'} />;
                            }}
                        />
                        <SubHeading>Whether or not to enable public notices for less sensitive log events.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'public_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Public log channel ID'} />;
                            }}
                        />
                        <SubHeading>Public log channel ID.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'public_match_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Public match log channel ID'} />;
                            }}
                        />
                        <SubHeading>
                            A channel to send match logs to. This can be very large and spammy, so its generally best to
                            use a separate channel, but not required.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'mod_ping_role_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Mod ping role ID'} />;
                            }}
                        />
                        <SubHeading>What role to include when pinging for certain events being sent.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'unregister_on_start'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return (
                                    <CheckboxSimple
                                        {...props}
                                        label={'Unregister existing discord slash commands on startup'}
                                    />
                                );
                            }}
                        />
                        <SubHeading>
                            Do you want to unregister all existing slash commands on bot startup. This is normally not
                            needed and its mostly useful when creating or modifying existing command.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
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

const LoggingSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, log: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            level: settings.log.level,
            file: settings.log.file,
            report_caller: settings.log.report_caller,
            full_timestamp: settings.log.full_timestamp
        }
    });

    return (
        <TabSection tab={'logging'} currentTab={tab} label={'Logging'} description={'Configure logger'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'level'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return (
                                    <SelectFieldSimple
                                        {...props}
                                        label={'Log Level'}
                                        items={['debug', 'info', 'warn', 'error']}
                                        renderMenu={(item) => {
                                            return (
                                                <MenuItem key={item} value={item}>
                                                    {item}
                                                </MenuItem>
                                            );
                                        }}
                                    />
                                );
                            }}
                        />
                        <SubHeading>What logging level to use.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'file'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Log file'} />;
                            }}
                        />
                        <SubHeading>If supplied, save log output to this file as well as stdout.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'report_caller'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Report caller'} />;
                            }}
                        />
                        <SubHeading>Enable/Disable logging of the function caller.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'full_timestamp'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Use full timestamp'} />;
                            }}
                        />
                        <SubHeading>Use full timestamps in the log output.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const SentrySection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, sentry: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            sentry_dsn: settings.sentry.sentry_dsn,
            sentry_dsn_web: settings.sentry.sentry_dsn_web,
            sentry_trace: settings.sentry.sentry_trace,
            sentry_sample_rate: settings.sentry.sentry_sample_rate
        }
    });

    return (
        <TabSection tab={'sentry'} currentTab={tab} label={'Sentry'} description={'Configure support for sentry'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        Self-hosted and cloud-based application performance monitoring & error tracking. You can create
                        a free account at <Link href={'https://sentry.io/'}>sentry.io</Link>. Otherwise you can follow
                        the guide for settings up{' '}
                        <Link href={'https://develop.sentry.dev/self-hosted/'}>self-hosted</Link> operation.
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'sentry_dsn'}
                            validators={{
                                onChange: z.string().refine(
                                    (arg) => {
                                        if (arg.length == 0) {
                                            return true;
                                        }

                                        return z.string().url().safeParse(arg).success;
                                    },
                                    { message: 'Invalid URL' }
                                )
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Backend sentry url'} />;
                            }}
                        />
                        <SubHeading>The URL to your backend sentry application.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'sentry_dsn_web'}
                            validators={{
                                onChange: z.string().refine(
                                    (arg) => {
                                        if (arg.length == 0) {
                                            return true;
                                        }

                                        return z.string().url().safeParse(arg).success;
                                    },
                                    { message: 'Invalid URL' }
                                )
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Frontend sentry url'} />;
                            }}
                        />
                        <SubHeading>The URL to your frontend sentry application.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'sentry_trace'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable tracing'} />;
                            }}
                        />
                        <SubHeading>
                            Enable <Link href={'https://docs.sentry.io/concepts/key-terms/tracing/'}>tracing</Link>{' '}
                            support.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'sentry_sample_rate'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 1))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Sample rate'} />;
                            }}
                        />
                        <SubHeading>
                            Configure the{' '}
                            <Link
                                href={
                                    'https://docs.sentry.io/platforms/go/configuration/sampling/#sampling-error-events'
                                }
                            >
                                sample rate
                            </Link>
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const GeoLocationSection = ({
    tab,
    settings,
    mutate
}: {
    tab: tabs;
    settings: Config;
    mutate: (s: Config) => void;
}) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, geo_location: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            enabled: settings.geo_location.enabled,
            cache_path: settings.geo_location.cache_path,
            token: settings.geo_location.token
        }
    });

    return (
        <TabSection
            tab={'geo_location'}
            currentTab={tab}
            label={'Geo Location'}
            description={'Configure ip2location integration'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        IP2Location is a 3rd party service that provides geoip databases along with some basic proxy
                        detections. gbans uses the IP2Location LITE database for{' '}
                        <Link href="https://lite.ip2location.com">IP geolocation</Link>. You must register for an
                        account to get an API key.
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable geolocation services'} />;
                            }}
                        />
                        <SubHeading>Enables the download and usage of geo location tools.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'token'}
                            validators={{
                                onChange: z.string().refine((arg) => arg.length == 0 || arg.length == 64)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'API Key'} />;
                            }}
                        />
                        <SubHeading>Your ip2location API key.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'cache_path'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Database download cache path'} />;
                            }}
                        />
                        <SubHeading>Path to store downloaded databases.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
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

const DebugSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, debug: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            skip_open_id_validation: settings.debug.skip_open_id_validation,
            add_rcon_log_address: settings.debug.add_rcon_log_address
        }
    });

    return (
        <TabSection
            tab={'debug'}
            currentTab={tab}
            label={'Debug'}
            description={'Configure debug options. Should not be used in production.'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'skip_open_id_validation'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Skip OpenID validation'} />;
                            }}
                        />
                        <SubHeading>
                            Disable validation for OpenID responses. Do not enable this on a live site.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
                        <Field
                            name={'add_rcon_log_address'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return (
                                    <TextFieldSimple
                                        {...props}
                                        label={'Extra log_address'}
                                        placeholder={'127.0.0.1:27115'}
                                    />
                                );
                            }}
                        />
                        <SubHeading>
                            Add this additional address to all known servers to start receiving log events. Make sure
                            you setup port forwarding.
                        </SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const LocalStoreSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, local_store: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            path_root: settings.local_store.path_root
        }
    });

    return (
        <TabSection
            tab={'local_store'}
            currentTab={tab}
            label={'Local Asset Store'}
            description={'Configure local asset storage'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'path_root'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path to store assets'} />;
                            }}
                        />
                        <SubHeading>Path to store all assets. Path is relative to gbans binary.</SubHeading>
                    </Grid>

                    <Grid xs={12}>
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

const SSHSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ssh: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            enabled: settings.ssh.enabled,
            username: settings.ssh.username,
            port: settings.ssh.port,
            private_key_path: settings.ssh.private_key_path,
            password: settings.ssh.password,
            update_interval: settings.ssh.update_interval,
            timeout: settings.ssh.timeout,
            demo_path_fmt: settings.ssh.demo_path_fmt
        }
    });

    return (
        <TabSection
            tab={'ssh'}
            currentTab={tab}
            label={'SSH/SCP Asset Fetching'}
            description={'Configure ssh settings for downloading demos'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable SSH downloader'} />;
                            }}
                        />
                        <SubHeading>Enable the use of SSH/SCP for downloading demos from a remote server.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'username'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH username'} />;
                            }}
                        />
                        <SubHeading>SSH username</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'port'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH port'} />;
                            }}
                        />
                        <SubHeading>
                            SSH port to use. This assumes all servers are configured using the same port.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'private_key_path'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path to private key'} />;
                            }}
                        />
                        <SubHeading>Path to your private key if using key based authentication.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'password'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH/Private key password'} />;
                            }}
                        />
                        <SubHeading>
                            Password when using standard auth. Passphrase to unlock the private key when using key auth.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'update_interval'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Check frequency (seconds)'} />;
                            }}
                        />
                        <SubHeading>How often to connect to remove systems and check for demos.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'timeout'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Connection timeout (seconds)'} />;
                            }}
                        />
                        <SubHeading>Connection timeout.</SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'demo_path_fmt'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path format for retrieving demos'} />;
                            }}
                        />
                        <SubHeading>
                            Format for generating a path to look for demos. Use <kbd>%s</kbd> as a substitution for the
                            short server name.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
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

const ExportsSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, exports: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            bd_enabled: settings.exports.bd_enabled,
            valve_enabled: settings.exports.valve_enabled
        }
    });

    return (
        <TabSection
            tab={'exports'}
            currentTab={tab}
            label={'Ban List Exports'}
            description={'Configure what is exported'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <Field
                            name={'bd_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={(props) => {
                                return (
                                    <CheckboxSimple {...props} label={'Enable tf2 bot detector compatible export'} />
                                );
                            }}
                        />
                        <SubHeading>
                            Enable exporting of a TF2 Bot Detector compatible player list. Only exports users banned
                            with the cheater reason.
                        </SubHeading>
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'valve_enabled'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <CheckboxSimple {...props} label={'Enable srcds formatted ban list'} />;
                            }}
                        />
                        <SubHeading>
                            Enable exporting of a SRCDS banned_user.cfg compatible player list. Only exports users
                            banned with the cheater reason.
                        </SubHeading>
                    </Grid>
                    {/*<Grid xs={12}>*/}
                    {/*    <Field*/}
                    {/*        name={'authorized_keys'}*/}
                    {/*        validators={{*/}
                    {/*            onChange: z.string()*/}
                    {/*        }}*/}
                    {/*        children={(props) => {*/}
                    {/*            return <TextFieldSimple {...props} label={'API Key'} />;*/}
                    {/*        }}*/}
                    {/*    />*/}
                    {/*</Grid>*/}

                    <Grid xs={12}>
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
