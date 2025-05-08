import { PropsWithChildren, ReactNode, useCallback, useState } from 'react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import BugReportIcon from '@mui/icons-material/BugReport';
import CleaningServicesIcon from '@mui/icons-material/CleaningServices';
import DeveloperBoardIcon from '@mui/icons-material/DeveloperBoard';
import EmergencyRecordingIcon from '@mui/icons-material/EmergencyRecording';
import GradingIcon from '@mui/icons-material/Grading';
import HeadsetMicIcon from '@mui/icons-material/HeadsetMic';
import LanIcon from '@mui/icons-material/Lan';
import PaymentIcon from '@mui/icons-material/Payment';
import SettingsIcon from '@mui/icons-material/Settings';
import ShareIcon from '@mui/icons-material/Share';
import TravelExploreIcon from '@mui/icons-material/TravelExplore';
import UpdateIcon from '@mui/icons-material/Update';
import WebAssetIcon from '@mui/icons-material/WebAsset';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { z } from 'zod';
import { apiGetDemoCleanup, apiGetNetworkUpdateDB } from '../api';
import { apiGetSettings, apiSaveSettings, Config } from '../api/admin.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
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
            'geo_location',
            'debug',
            'local_store',
            'network',
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
    | 'geo_location'
    | 'debug'
    | 'network'
    | 'local_store'
    | 'ssh'
    | 'exports';

type TabButtonProps<Tabs> = {
    label: string;
    tab: Tabs;
    onClick: (tab: Tabs) => void;
    currentTab: Tabs;
    icon: ReactNode;
};

export const TabButton = <Tabs,>({ currentTab, tab, label, onClick, icon }: TabButtonProps<Tabs>) => {
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

export const TabSection = <Tabs,>({
    children,
    tab,
    currentTab,
    label,
    description
}: PropsWithChildren & {
    tab: Tabs;
    currentTab: Tabs;
    label: string;
    description: string;
}) => {
    return (
        <Grid size={{ xs: 8, sm: 9, md: 10 }} display={tab == currentTab ? undefined : 'none'} marginTop={1}>
            <Typography variant={'h1'}>{label}</Typography>
            <Typography variant={'subtitle1'} marginBottom={2}>
                {description}
            </Typography>
            {children}
        </Grid>
    );
};

const ConfigContainer = ({ children }: { children: ReactNode[] }) => {
    return (
        <Grid container spacing={4}>
            {children}
        </Grid>
    );
};

export const SubHeading = ({ children }: PropsWithChildren) => (
    <Typography variant={'subtitle1'} padding={1}>
        {children}
    </Typography>
);

function AdminServers() {
    const { sendFlash, sendError } = useUserFlashCtx();
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
        onError: sendError
    });

    const onTabClick = useCallback(
        async (section: tabs) => {
            setTab(section);
            await navigate({ to: '/admin/settings', replace: true, search: { section } });
        },
        [setTab]
    );

    return (
        <>
            <Title>Edit Settings</Title>

            <ContainerWithHeaderAndButtons title={'System Settings'} iconLeft={<DeveloperBoardIcon />}>
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
                                tab={'filters'}
                                onClick={onTabClick}
                                icon={<AddModeratorIcon />}
                                currentTab={tab}
                                label={'Filters'}
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
                                tab={'geo_location'}
                                onClick={onTabClick}
                                icon={<TravelExploreIcon />}
                                currentTab={tab}
                                label={'GeoDB'}
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
                                tab={'network'}
                                onClick={onTabClick}
                                icon={<LanIcon />}
                                currentTab={tab}
                                label={'Network'}
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
                    <GeoLocationSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <LocalStoreSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <NetworkSection tab={tab} settings={settings} mutate={mutation.mutate} />
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
        defaultValues: {
            srcds_log_addr: settings.general.srcds_log_addr,
            file_serve_mode: settings.general.file_serve_mode,
            mode: settings.general.mode,
            site_name: settings.general.site_name,
            asset_url: settings.general.asset_url,
            default_route: settings.general.default_route,
            news_enabled: settings.general.news_enabled,
            forums_enabled: settings.general.forums_enabled,
            contests_enabled: settings.general.contests_enabled,
            wiki_enabled: settings.general.wiki_enabled,
            stats_enabled: settings.general.stats_enabled,
            servers_enabled: settings.general.servers_enabled,
            reports_enabled: settings.general.reports_enabled,
            chatlogs_enabled: settings.general.chatlogs_enabled,
            demos_enabled: settings.general.demos_enabled,
            speedruns_enabled: settings.general.speedruns_enabled,
            playerqueue_enabled: settings.general.playerqueue_enabled
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This name is displayed in various places throughout the app such as the title bar and site
                            heading. It should be short and simple.
                        </SubHeading>
                        <Field
                            name={'site_name'}
                            validators={{
                                onChange: z.string().min(1).max(32)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Global Site Name'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If you have a asset under a different subdir you should change this.</SubHeading>
                        <Field
                            name={'asset_url'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'URL path pointing to assets'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            What address to listen for UDP log events. host:port format. If host is empty, it will
                            listen on all available hosts.
                        </SubHeading>
                        <Field
                            name={'srcds_log_addr'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'UDP Log Listen Address'} />;
                            }}
                        />
                    </Grid>

                    <Typography variant={'h5'}>Configure Toplevel Features</Typography>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Sets the default page to load when a user opens the root url <kbd>example.com/</kbd>.
                        </SubHeading>
                        <Field
                            name={'default_route'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Default Index Route'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable the news/blog functionality.</SubHeading>
                        <Field
                            name={'news_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable news features.'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled/disable the forums functionality.</SubHeading>
                        <Field
                            name={'forums_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable forums'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable contests in which users can participate. Users can submit entries and users can vote
                            on them.
                        </SubHeading>
                        <Field
                            name={'contests_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable contests'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables a wiki section which is editable by moderators, and viewable by the public.
                        </SubHeading>
                        <Field
                            name={'wiki_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Wiki'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Allows users to search and download demos.</SubHeading>
                        <Field
                            name={'demos_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Demo/STV Support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Process demos and calculate game stats.</SubHeading>
                        <Field
                            name={'stats_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Game Stats'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables the server status page showing the current map and player counts.
                        </SubHeading>
                        <Field
                            name={'servers_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Servers Page'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Allows users to report other users.</SubHeading>
                        <Field
                            name={'reports_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable User Reports'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable showing the searchable chatlogs.</SubHeading>
                        <Field
                            name={'chatlogs_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable public chatlogs'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables the 1000 uncles speedruns tracking support.</SubHeading>
                        <Field
                            name={'speedruns_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Speedruns support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables the functionality allowing players to queue up together using the website.
                        </SubHeading>
                        <Field
                            name={'playerqueue_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Playerqueue support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const NetworkSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, network: value });
        },
        defaultValues: {
            sdr_enabled: settings.network.sdr_enabled,
            sdr_dns_enabled: settings.network.sdr_dns_enabled,
            cf_key: settings.network.cf_key,
            cf_email: settings.network.cf_email
        }
    });

    return (
        <TabSection
            tab={'network'}
            currentTab={tab}
            label={'Network'}
            description={'Advanced Networking Functionality'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <Typography variant={'h3'}>Configure SDR Features</Typography>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable SDR (Steam Data Relay)</SubHeading>
                        <Field
                            name={'sdr_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable SDR networking mode'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If you have a asset under a different subdir you should change this.</SubHeading>
                        <Field
                            name={'sdr_dns_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable SDR DNS updates'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Typography variant={'h3'}>Configure DNS Providers</Typography>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Typography variant={'h5'}>Configure DNS Providers</Typography>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Your API key created on cloudflare. This key must have DNS editing privileges.
                        </SubHeading>
                        <Field
                            name={'cf_key'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'API Key'} />;
                            }}
                        />
                    </Grid>

                    <Typography variant={'h5'}>Configure Toplevel Features</Typography>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Sets the default page to load when a user opens the root url <kbd>example.com/</kbd>.
                        </SubHeading>
                        <Field
                            name={'cf_email'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Email'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const FiltersSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, filters: value });
        },
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable/disable the feature</SubHeading>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Word Filters'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If a user gets a warning, it will expire after this duration of time.</SubHeading>
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
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        {' '}
                        <SubHeading>
                            A hard limit to the number of warnings a user can receive before action is taken.
                        </SubHeading>
                        <Field
                            name={'warning_limit'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 1000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Maximum number of warnings allowed'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Run the chat filters, but do not actually punish users.</SubHeading>
                        <Field
                            name={'dry'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable dry run mode'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If discord is enabled, send filter match notices to the log channel.</SubHeading>
                        <Field
                            name={'ping_discord'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Send discord notices on match'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            When the sum of warning weights issued to a user is greater than this value, take action
                            against the user.
                        </SubHeading>
                        <Field
                            name={'max_weight'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(1, 1000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Max Weight'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How frequent warnings will be checked for users exceeding limits.</SubHeading>
                        <Field
                            name={'check_timeout'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(5, 300))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Check Frequency (seconds)'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How long it takes for a users warning to expire after being matched.</SubHeading>
                        <Field
                            name={'match_timeout'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(1, 10000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Match Timeout'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const DemosSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const queryClient = useQueryClient();
    const { sendFlash } = useUserFlashCtx();

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, demo: value });
        },
        defaultValues: {
            demo_cleanup_enabled: settings.demo.demo_cleanup_enabled,
            demo_cleanup_strategy: settings.demo.demo_cleanup_strategy ?? 'pctfree',
            demo_cleanup_min_pct: settings.demo.demo_cleanup_min_pct,
            demo_cleanup_mount: settings.demo.demo_cleanup_mount,
            demo_count_limit: settings.demo.demo_count_limit,
            demo_parser_url: settings.demo.demo_parser_url
        }
    });

    const onCleanup = useCallback(async () => {
        try {
            await queryClient.fetchQuery({ queryKey: ['demoCleanup'], queryFn: apiGetDemoCleanup });
            sendFlash('success', 'Cleanup started');
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Cleanup failed to start');
        }
    }, []);

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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <Button
                            startIcon={<CleaningServicesIcon />}
                            variant={'contained'}
                            color={'secondary'}
                            onClick={onCleanup}
                        >
                            Start Cleanup
                        </Button>
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable automatic deletion of demos. This ignores demos that have been marked as archived.
                        </SubHeading>
                        <Field
                            name={'demo_cleanup_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Scheduled Demo Cleanup'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Method used to determine what demos to delete.</SubHeading>
                        <Field
                            name={'demo_cleanup_strategy'}
                            validators={{
                                onChange: z.enum(['pctfree', 'count'])
                            }}
                            children={(props) => {
                                return (
                                    <SelectFieldSimple
                                        {...props}
                                        value={props.state.value}
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
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            When using the percent free strategy, defined how much free space should be retained on the
                            demo mount/volume.
                        </SubHeading>
                        <Field
                            name={'demo_cleanup_min_pct'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 100))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Minimum percent free space to retain'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>The mount point that demos are stored. Used to determine free space.</SubHeading>
                        <Field
                            name={'demo_cleanup_mount'}
                            validators={{
                                onChange: z.string().startsWith('/')
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Mount point to check for free space'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            When using the count deletion strategy, this is the maximum number of demos to keep.
                        </SubHeading>
                        <Field
                            name={'demo_count_limit'}
                            validators={{
                                onChange: z.string().transform(numberStringValidator(0, 100000))
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Max amount of demos to keep'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This url should point to an instance of https://github.com/leighmacdonald/tf2_demostats.
                            This is used to pull stats & player steamids out of demos that are fetched.
                        </SubHeading>
                        <Field
                            name={'demo_parser_url'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'URL for demo parsing submissions'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const PatreonSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, patreon: value });
        },
        defaultValues: {
            enabled: settings.patreon.enabled,
            integrations_enabled: settings.patreon.integrations_enabled,
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
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled/Disable patreon integrations</SubHeading>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable Patreon Integration'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables integration into the website. Enables: Donate button, Account Linking.
                        </SubHeading>
                        <Field
                            name={'integrations_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable website integrations'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your patron client ID</SubHeading>
                        <Field
                            name={'client_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Client ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Patreon app client secret</SubHeading>
                        <Field
                            name={'client_secret'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Client Secret'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Access token</SubHeading>
                        <Field
                            name={'creator_access_token'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Access Token'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Refresh token</SubHeading>
                        <Field
                            name={'creator_refresh_token'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Refresh Token'} />;
                            }}
                        />
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

const DiscordSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, discord: value });
        },
        defaultValues: {
            enabled: settings.discord.enabled,
            bot_enabled: settings.discord.bot_enabled,
            integrations_enabled: settings.discord.integrations_enabled,
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
            vote_log_channel_id: settings.discord.vote_log_channel_id,
            appeal_log_channel_id: settings.discord.appeal_log_channel_id,
            ban_log_channel_id: settings.discord.ban_log_channel_id,
            forum_log_channel_id: settings.discord.forum_log_channel_id,
            word_filter_log_channel_id: settings.discord.word_filter_log_channel_id,
            kick_log_channel_id: settings.discord.kick_log_channel_id,
            playerqueue_channel_id: settings.discord.playerqueue_channel_id,
            anticheat_channel_id: settings.discord.anticheat_channel_id
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled or disable all discord integration.</SubHeading>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable discord integration'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enabled the discord bot integration. This is self-hosted within the app. You must can create
                            a discord application{' '}
                            <Link href={'https://discord.com/developers/applications?new_application=true'}>here</Link>.
                        </SubHeading>
                        <Field
                            name={'bot_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Discord Bot'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables integrations into the website. Enables: Showing Join Discord button, Account
                            Linking.
                        </SubHeading>
                        <Field
                            name={'integrations_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable website integrations'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your discord application ID.</SubHeading>
                        <Field
                            name={'app_id'}
                            validators={{
                                onChange: z.string().refine((arg) => arg.length == 0 || arg.length == 18)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord app ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your discord app secret.</SubHeading>
                        <Field
                            name={'app_secret'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord bot app secret'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            The unique ID for your permanent discord link. This is only the unique string at the end if
                            a invite url: https://discord.gg/&lt;XXXXXXXXX&gt;, not the entire url.
                        </SubHeading>
                        <Field
                            name={'link_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Invite link ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Bot authentication token.</SubHeading>
                        <Field
                            name={'token'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord Bot Token'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This is the guild id of your discord server. With discoed developer mode enabled,
                            right-click on the server title and select "Copy ID" to get the guild ID.
                        </SubHeading>
                        <Field
                            name={'guild_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Discord guild ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This should be a private channel. Its the default log channel and is used as the default for
                            other channels if their id is empty.
                        </SubHeading>
                        <Field
                            name={'log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <Field
                            name={'public_log_channel_enable'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable public log channel'} />;
                            }}
                        />
                        <SubHeading>Whether or not to enable public notices for less sensitive log events.</SubHeading>
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What role to include when pinging for certain events being sent.</SubHeading>
                        <Field
                            name={'mod_ping_role_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Mod ping role ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Public log channel ID.</SubHeading>
                        <Field
                            name={'public_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Public log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send match logs to. This can be very large and spammy, so its generally best to
                            use a separate channel, but not required.
                        </SubHeading>
                        <Field
                            name={'public_match_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Public match log channel ID'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send in-game kick voting. This can be very noisy, so its generally best to use
                            a separate channel, but not required.
                        </SubHeading>
                        <Field
                            name={'vote_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Vote log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>New appeals and appeal messages are shown here.</SubHeading>
                        <Field
                            name={'appeal_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Appeal changelog channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send match logs to. This can be very large and spammy, so its generally best to
                            use a separate channel, but not required. This only shows steam based bans.
                        </SubHeading>
                        <Field
                            name={'ban_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'New ban log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Show new forum activity. This includes threads, new messages, message deletions.
                        </SubHeading>
                        <Field
                            name={'forum_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Forum activity log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>A channel to send notices to when a user triggers a word filter.</SubHeading>
                        <Field
                            name={'word_filter_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Word filter log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send notices to when a user is kicked either from being banned or denied entry
                            while already in a banned state.
                        </SubHeading>
                        <Field
                            name={'kick_log_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Kick log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>A channel which relays the chat messages from the website chat lobby.</SubHeading>
                        <Field
                            name={'playerqueue_channel_id'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Playerqueue log channel ID'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const LoggingSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, log: value });
        },
        defaultValues: {
            level: settings.log.level ?? 'info',
            file: settings.log.file,
            http_enabled: settings.log.http_enabled,
            http_otel_enabled: settings.log.http_otel_enabled,
            http_level: settings.log.http_level
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What logging level to use.</SubHeading>
                        <Field
                            name={'level'}
                            validators={{
                                onChange: z.enum(['debug', 'info', 'warn', 'error'])
                            }}
                            children={(props) => {
                                return (
                                    <SelectFieldSimple
                                        {...props}
                                        value={props.state.value}
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
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If supplied, save log output to this file as well as stdout.</SubHeading>
                        <Field
                            name={'file'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Log file'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables logging for incoming HTTP requests.</SubHeading>
                        <Field
                            name={'http_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable HTTP request logs'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables OpenTelemetry support (span id/trace id).</SubHeading>
                        <Field
                            name={'http_otel_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable OpenTelemetry Support'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What logging level to use for HTTP requests.</SubHeading>
                        <Field
                            name={'http_level'}
                            validators={
                                {
                                    //onChange: z.string()
                                }
                            }
                            children={(props) => {
                                return (
                                    <SelectFieldSimple
                                        {...props}
                                        label={'HTTP Log Level'}
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
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
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
    const { sendFlash } = useUserFlashCtx();
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, geo_location: value });
        },
        defaultValues: {
            enabled: settings.geo_location.enabled,
            cache_path: settings.geo_location.cache_path,
            token: settings.geo_location.token
        }
    });

    const onUpdateDB = useCallback(async () => {
        try {
            await apiGetNetworkUpdateDB();
            sendFlash('success', 'Started database update');
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Update already running');
        }
    }, [sendFlash]);

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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        IP2Location is a 3rd party service that provides geoip databases along with some basic proxy
                        detections. gbans uses the IP2Location LITE database for{' '}
                        <Link href="https://lite.ip2location.com">IP geolocation</Link>. You must register for an
                        account to get an API key.
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <Button variant={'contained'} startIcon={<UpdateIcon />} onClick={onUpdateDB}>
                            Update Database
                        </Button>
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables the download and usage of geo location tools.</SubHeading>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable geolocation services'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your ip2location API key.</SubHeading>
                        <Field
                            name={'token'}
                            validators={{
                                onChange: z.string().refine((arg) => arg.length == 0 || arg.length == 64)
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'API Key'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Path to store downloaded databases.</SubHeading>
                        <Field
                            name={'cache_path'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Database download cache path'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const DebugSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, debug: value });
        },
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Disable validation for OpenID responses. Do not enable this on a live site.
                        </SubHeading>
                        <Field
                            name={'skip_open_id_validation'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Skip OpenID validation'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Add this additional address to all known servers to start receiving log events. Make sure
                            you setup port forwarding.
                        </SubHeading>
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
                                        placeholder={'127.0.0.1:27715'}
                                    />
                                );
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const LocalStoreSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, local_store: value });
        },
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
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

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const SSHSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ssh: value });
        },
        defaultValues: {
            enabled: settings.ssh.enabled,
            username: settings.ssh.username,
            port: settings.ssh.port,
            private_key_path: settings.ssh.private_key_path,
            password: settings.ssh.password,
            update_interval: settings.ssh.update_interval,
            timeout: settings.ssh.timeout,
            demo_path_fmt: settings.ssh.demo_path_fmt,
            stac_path_fmt: settings.ssh.stac_path_fmt
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable the use of SSH/SCP for downloading demos from a remote server.</SubHeading>
                        <Field
                            name={'enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable SSH downloader'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>SSH username</SubHeading>
                        <Field
                            name={'username'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH username'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            SSH port to use. This assumes all servers are configured using the same port.
                        </SubHeading>
                        <Field
                            name={'port'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH port'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Path to your private key if using key based authentication.</SubHeading>
                        <Field
                            name={'private_key_path'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path to private key'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Password when using standard auth. Passphrase to unlock the private key when using key auth.
                        </SubHeading>
                        <Field
                            name={'password'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'SSH/Private key password'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How often to connect to remove systems and check for demos.</SubHeading>
                        <Field
                            name={'update_interval'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Check frequency (seconds)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Connection timeout.</SubHeading>
                        <Field
                            name={'timeout'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Connection timeout (seconds)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Format for generating a path to look for demos. Use <kbd>%s</kbd> as a substitution for the
                            short server name.
                        </SubHeading>
                        <Field
                            name={'demo_path_fmt'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path format for retrieving demos'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Format for generating a path to look for stac anticheat logs. Use <kbd>%s</kbd> as a
                            substitution for the short server name.
                        </SubHeading>
                        <Field
                            name={'stac_path_fmt'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Path format for retrieving stac logs'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const ExportsSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, exports: value });
        },
        defaultValues: {
            bd_enabled: settings.exports.bd_enabled,
            valve_enabled: settings.exports.valve_enabled,
            authorized_keys: settings.exports.authorized_keys
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
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Comma separated list of authorized keys which can access these resources. If no keys are
                            specified, access will be granted to everyone. Append key to query with{' '}
                            <kbd>&key=value</kbd>
                        </SubHeading>
                        <Field
                            name={'authorized_keys'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} label={'Authorized Keys (comma separated).'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable exporting of a TF2 Bot Detector compatible player list. Only exports users banned
                            with the cheater reason.
                        </SubHeading>
                        <Field
                            name={'bd_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable tf2 bot detector compatible export'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable exporting of a SRCDS banned_user.cfg compatible player list. Only exports users
                            banned with the cheater reason.
                        </SubHeading>
                        <Field
                            name={'valve_enabled'}
                            validators={{
                                onChange: z.boolean()
                            }}
                            children={() => {
                                return <CheckboxSimple label={'Enable srcds formatted ban list'} />;
                            }}
                        />
                    </Grid>
                    {/*<Grid size={{xs: 12}}>*/}
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

                    <Grid size={{ xs: 12 }}>
                        <Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                            )}
                        />
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};
