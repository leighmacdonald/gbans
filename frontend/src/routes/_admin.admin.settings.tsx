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
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { z } from 'zod';
import { apiGetDemoCleanup, apiGetNetworkUpdateDB } from '../api';
import { apiGetSettings, apiSaveSettings } from '../api/admin.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { Title } from '../component/Title';
import { CheckboxField } from '../component/form/field/CheckboxField.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import {
    Config,
    schemaDebug,
    schemaDemos,
    schemaDiscord,
    schemaExports,
    schemaFilters,
    schemaGeneral,
    schemaGeo,
    schemaLocalStore,
    schemaLogging,
    schemaNetwork,
    schemaPatreon,
    schemaSSH
} from '../schema/config.ts';
import { logErr } from '../util/errors.ts';

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
    component: AdminSettings,
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

function AdminSettings() {
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
    const defaultValues: z.input<typeof schemaGeneral> = settings.general;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, general: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaGeneral
        }
    });

    return (
        <TabSection
            tab={'general'}
            currentTab={tab}
            label={'General'}
            description={'Core settings that do not fit into a subcategory'}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This name is displayed in various places throughout the app such as the title bar and site
                            heading. It should be short and simple.
                        </SubHeading>
                        <form.AppField
                            name={'site_name'}
                            children={(field) => {
                                return <field.TextField label={'Global Site Name'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If you have an asset under a different subdir you should change this.</SubHeading>
                        <form.AppField
                            name={'asset_url'}
                            children={(field) => {
                                return <field.TextField label={'URL path pointing to assets'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            What address to listen for UDP log events. host:port format. If host is empty, it will
                            listen on all available hosts.
                        </SubHeading>
                        <form.AppField
                            name={'srcds_log_addr'}
                            children={(field) => {
                                return <field.TextField label={'UDP Log Listen Address'} />;
                            }}
                        />
                    </Grid>

                    <Typography variant={'h5'}>Configure Toplevel Features</Typography>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Sets the default page to load when a user opens the root url <kbd>example.com/</kbd>.
                        </SubHeading>
                        <form.AppField
                            name={'default_route'}
                            children={(field) => {
                                return <field.TextField label={'Default Index Route'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable the news/blog functionality.</SubHeading>
                        <form.AppField
                            name={'news_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable news features.'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled/disable the forums functionality.</SubHeading>
                        <form.AppField
                            name={'forums_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable forums'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable contests in which users can participate. Users can submit entries and users can vote
                            on them.
                        </SubHeading>
                        <form.AppField
                            name={'contests_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable contests'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables a wiki section which is editable by moderators, and viewable by the public.
                        </SubHeading>
                        <form.AppField
                            name={'wiki_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Wiki'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Allows users to search and download demos.</SubHeading>
                        <form.AppField
                            name={'demos_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Demo/STV Support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Process demos and calculate game stats.</SubHeading>
                        <form.AppField
                            name={'stats_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Game Stats'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables the server status page showing the current map and player counts.
                        </SubHeading>
                        <form.AppField
                            name={'servers_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Servers Page'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Allows users to report other users.</SubHeading>
                        <form.AppField
                            name={'reports_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable User Reports'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable showing the searchable chatlogs.</SubHeading>
                        <form.AppField
                            name={'chatlogs_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable public chatlogs'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables the 1000 uncles speedruns tracking support.</SubHeading>
                        <form.AppField
                            name={'speedruns_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Speedruns support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables the functionality allowing players to queue up together using the website.
                        </SubHeading>
                        <form.AppField
                            name={'playerqueue_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Playerqueue support'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const NetworkSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaNetwork> = settings.network;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, network: value });
        },
        defaultValues,
        validators: {
            onChange: schemaNetwork
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
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <Typography variant={'h3'}>Steam Datagram Relay</Typography>
                        <Typography variant={'body1'}>
                            Steam Datagram Relay (SDR) is Valve's virtual private gaming network.
                        </Typography>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable SDR (Steam Data Relay)</SubHeading>
                        <form.AppField
                            name={'sdr_enabled'}
                            children={() => {
                                return <CheckboxField label={'Enable SDR networking mode'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If you have an asset under a different subdir you should change this.</SubHeading>
                        <form.AppField
                            name={'sdr_dns_enabled'}
                            children={() => {
                                return <CheckboxField label={'Enable SDR DNS updates'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <Typography variant={'h3'}>Cloudflare</Typography>
                        <Typography variant={'body1'}>
                            Current cloudflare is the only supported DNS provider. If you want to see others added, feel
                            free to open a GitHub issue.
                        </Typography>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading></SubHeading>
                        <form.AppField
                            name={'cf_key'}
                            children={(field) => {
                                return (
                                    <field.TextField
                                        label={'API Key'}
                                        type={'password'}
                                        helperText={
                                            'Your API key created on cloudflare. This key must have DNS editing privileges.'
                                        }
                                    />
                                );
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Account Email Address</SubHeading>
                        <form.AppField
                            name={'cf_email'}
                            children={(field) => {
                                return <field.TextField label={'Email'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Zone ID for the domain.</SubHeading>
                        <form.AppField
                            name={'cf_zone_id'}
                            children={(field) => {
                                return <field.TextField label={'Email'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const FiltersSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaFilters> = settings.filters;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, filters: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaFilters
        }
    });

    return (
        <TabSection
            tab={'filters'}
            currentTab={tab}
            label={'Word Filters'}
            description={
                'Word filters are a form of auto-moderation that scans ' +
                'incoming chat logs and usernames for matching values and handles them accordingly'
            }
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable/disable the feature</SubHeading>
                        <form.AppField
                            name={'enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Word Filters'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If a user gets a warning, it will expire after this duration of time.</SubHeading>
                        <form.AppField
                            name={'warning_timeout'}
                            children={(field) => {
                                return <field.TextField label={'How long until a warning expires (seconds)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        {' '}
                        <SubHeading>
                            A hard limit to the number of warnings a user can receive before action is taken.
                        </SubHeading>
                        <form.AppField
                            name={'warning_limit'}
                            children={(field) => {
                                return <field.TextField label={'Maximum number of warnings allowed'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Run the chat filters, but do not actually punish users.</SubHeading>
                        <form.AppField
                            name={'dry'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable dry run mode'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>If discord is enabled, send filter match notices to the log channel.</SubHeading>
                        <form.AppField
                            name={'ping_discord'}
                            children={(field) => {
                                return <field.CheckboxField label={'Send discord notices on match'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            When the sum of warning weights issued to a user is greater than this value, take action
                            against the user.
                        </SubHeading>
                        <form.AppField
                            name={'max_weight'}
                            children={(field) => {
                                return <field.TextField label={'Max Weight'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How frequent warnings will be checked for users exceeding limits.</SubHeading>
                        <form.AppField
                            name={'check_timeout'}
                            children={(field) => {
                                return <field.TextField label={'Check Frequency (seconds)'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How long it takes for a users warning to expire after being matched.</SubHeading>
                        <form.AppField
                            name={'match_timeout'}
                            children={(field) => {
                                return <field.TextField label={'Match Timeout'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const DemosSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaDemos> = settings.demo;
    const queryClient = useQueryClient();
    const { sendFlash } = useUserFlashCtx();

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, demo: value });
        },
        defaultValues: defaultValues,
        validators: {
            onSubmit: schemaDemos
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
                    await form.handleSubmit();
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
                        <form.AppField
                            name={'demo_cleanup_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Scheduled Demo Cleanup'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Method used to determine what demos to delete.</SubHeading>
                        <form.AppField
                            name={'demo_cleanup_strategy'}
                            children={(field) => {
                                return (
                                    <field.SelectField
                                        label={'Cleanup Strategy'}
                                        items={['pctfree', 'count']}
                                        renderItem={(item) => {
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
                        <form.AppField
                            name={'demo_cleanup_min_pct'}
                            children={(field) => {
                                return <field.TextField label={'Minimum percent free space to retain'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>The mount point that demos are stored. Used to determine free space.</SubHeading>
                        <form.AppField
                            name={'demo_cleanup_mount'}
                            children={(field) => {
                                return <field.TextField label={'Mount point to check for free space'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            When using the count deletion strategy, this is the maximum number of demos to keep.
                        </SubHeading>
                        <form.AppField
                            name={'demo_count_limit'}
                            children={(field) => {
                                return <field.TextField label={'Max amount of demos to keep'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This url should point to an instance of https://github.com/leighmacdonald/tf2_demostats.
                            This is used to pull stats & player steamids out of demos that are fetched.
                        </SubHeading>
                        <form.AppField
                            name={'demo_parser_url'}
                            children={(field) => {
                                return <field.TextField label={'URL for demo parsing submissions'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const PatreonSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaPatreon> = settings.patreon;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, patreon: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaPatreon
        }
    });

    return (
        <TabSection tab={'patreon'} currentTab={tab} label={'Patreon'} description={'Connect to patreon API'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled/Disable patreon integrations</SubHeading>
                        <form.AppField
                            name={'enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable Patreon Integration'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables integration into the website. Enables: Donate button, Account Linking.
                        </SubHeading>
                        <form.AppField
                            name={'integrations_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable website integrations'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your patron client ID</SubHeading>
                        <form.AppField
                            name={'client_id'}
                            children={(field) => {
                                return <field.TextField label={'Client ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Patreon app client secret</SubHeading>
                        <form.AppField
                            name={'client_secret'}
                            children={(field) => {
                                return <field.TextField label={'Client Secret'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Access token</SubHeading>
                        <form.AppField
                            name={'creator_access_token'}
                            children={(field) => {
                                return <field.TextField label={'Access Token'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Refresh token</SubHeading>
                        <form.AppField
                            name={'creator_refresh_token'}
                            children={(field) => {
                                return <field.TextField label={'Refresh Token'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </Grid>
            </form>
        </TabSection>
    );
};

const DiscordSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaDiscord> = settings.discord;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, discord: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaDiscord
        }
    });

    return (
        <TabSection tab={'discord'} currentTab={tab} label={'Discord'} description={'Support for discord bot'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enabled or disable all discord integration.</SubHeading>
                        <form.AppField
                            name={'enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable discord integration'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enabled the discord bot integration. This is self-hosted within the app. You can create a
                            discord application{' '}
                            <Link href={'https://discord.com/developers/applications?new_application=true'}>here</Link>.
                        </SubHeading>
                        <form.AppField
                            name={'bot_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Discord Bot'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enables integrations into the website. Enables: Showing Join Discord button, Account
                            Linking.
                        </SubHeading>
                        <form.AppField
                            name={'integrations_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable website integrations'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your discord application ID.</SubHeading>
                        <form.AppField
                            name={'app_id'}
                            children={(field) => {
                                return <field.TextField label={'Discord app ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your discord app secret.</SubHeading>
                        <form.AppField
                            name={'app_secret'}
                            children={(field) => {
                                return <field.TextField label={'Discord bot app secret'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            The unique ID for your permanent discord link. This is only the unique string at the end if
                            an invitation url: https://discord.gg/&lt;XXXXXXXXX&gt;, not the entire url.
                        </SubHeading>
                        <form.AppField
                            name={'link_id'}
                            children={(field) => {
                                return <field.TextField label={'Invite link ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Bot authentication token.</SubHeading>
                        <form.AppField
                            name={'token'}
                            children={(field) => {
                                return <field.TextField label={'Discord Bot Token'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This is the guild id of your discord server. With discoed developer mode enabled,
                            right-click on the server title and select "Copy ID" to get the guild ID.
                        </SubHeading>
                        <form.AppField
                            name={'guild_id'}
                            children={(field) => {
                                return <field.TextField label={'Discord guild ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            This should be a private channel. It's the default log channel and is used as the default
                            for other channels if their id is empty.
                        </SubHeading>
                        <form.AppField
                            name={'log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.AppField
                            name={'public_log_channel_enable'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable public log channel'} />;
                            }}
                        />
                        <SubHeading>Whether or not to enable public notices for less sensitive log events.</SubHeading>
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What role to include when pinging for certain events being sent.</SubHeading>
                        <form.AppField
                            name={'mod_ping_role_id'}
                            children={(field) => {
                                return <field.TextField label={'Mod ping role ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Public log channel ID.</SubHeading>
                        <form.AppField
                            name={'public_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Public log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send match logs to. This can be very large and spammy, so it's generally best
                            to use a separate channel, but not required.
                        </SubHeading>
                        <form.AppField
                            name={'public_match_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Public match log channel ID'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send in-game kick voting. This can be very noisy, so it's generally best to use
                            a separate channel, but not required.
                        </SubHeading>
                        <form.AppField
                            name={'vote_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Vote log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>New appeals and appeal messages are shown here.</SubHeading>
                        <form.AppField
                            name={'appeal_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Appeal changelog channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send match logs to. This can be very large and spammy, so it's generally best
                            to use a separate channel, but not required. This only shows steam based bans.
                        </SubHeading>
                        <form.AppField
                            name={'ban_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'New ban log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Show new forum activity. This includes threads, new messages, message deletions.
                        </SubHeading>
                        <form.AppField
                            name={'forum_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Forum activity log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>A channel to send notices to when a user triggers a word filter.</SubHeading>
                        <form.AppField
                            name={'word_filter_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Word filter log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            A channel to send notices to when a user is kicked either from being banned or denied entry
                            while already in a banned state.
                        </SubHeading>
                        <form.AppField
                            name={'kick_log_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Kick log channel ID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>A channel which relays the chat messages from the website chat lobby.</SubHeading>
                        <form.AppField
                            name={'playerqueue_channel_id'}
                            children={(field) => {
                                return <field.TextField label={'Playerqueue log channel ID'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const LoggingSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaLogging> = settings.log;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, log: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaLogging
        }
    });

    return (
        <TabSection tab={'logging'} currentTab={tab} label={'Logging'} description={'Configure logger'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What logging level to use.</SubHeading>
                        <form.AppField
                            name={'level'}
                            children={(field) => {
                                return (
                                    <field.SelectField
                                        label={'Log Level'}
                                        items={['debug', 'info', 'warn', 'error']}
                                        renderItem={(item) => {
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
                        <form.AppField
                            name={'file'}
                            children={(field) => {
                                return <field.TextField label={'Log file'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables logging for incoming HTTP requests.</SubHeading>
                        <form.AppField
                            name={'http_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable HTTP request logs'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enables OpenTelemetry support (span id/trace id).</SubHeading>
                        <form.AppField
                            name={'http_otel_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable OpenTelemetry Support'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>What logging level to use for HTTP requests.</SubHeading>
                        <form.AppField
                            name={'http_level'}
                            children={(field) => {
                                return (
                                    <field.SelectField
                                        label={'HTTP Log Level'}
                                        items={['debug', 'info', 'warn', 'error']}
                                        renderItem={(item) => {
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
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
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
    const defaultValues: z.input<typeof schemaGeo> = settings.geo_location;
    const { sendFlash } = useUserFlashCtx();
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, geo_location: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaGeo
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
                    await form.handleSubmit();
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
                        <SubHeading>Enables the download and usage of geolocation tools.</SubHeading>
                        <form.AppField
                            name={'enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable geolocation services'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Your ip2location API key.</SubHeading>
                        <form.AppField
                            name={'token'}
                            children={(field) => {
                                return <field.TextField label={'API Key'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Path to store downloaded databases.</SubHeading>
                        <form.AppField
                            name={'cache_path'}
                            children={(field) => {
                                return <field.TextField label={'Database download cache path'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const DebugSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaDebug> = settings.debug;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, debug: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaDebug
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
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Disable validation for OpenID responses. Do not enable this on a live site.
                        </SubHeading>
                        <form.AppField
                            name={'skip_open_id_validation'}
                            children={(field) => {
                                return <field.CheckboxField label={'Skip OpenID validation'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Add this additional address to all known servers to start receiving log events. Make sure
                            you set up port forwarding.
                        </SubHeading>
                        <form.AppField
                            name={'add_rcon_log_address'}
                            children={(field) => {
                                return <field.TextField label={'Extra log_address'} placeholder={'127.0.0.1:27715'} />;
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const LocalStoreSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaLocalStore> = settings.local_store;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, local_store: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaLocalStore
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
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <form.AppField
                            name={'path_root'}
                            children={(field) => {
                                return <field.TextField label={'Path to store assets'} />;
                            }}
                        />
                        <SubHeading>Path to store all assets. Path is relative to gbans binary.</SubHeading>
                    </Grid>

                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const SSHSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaSSH> = settings.ssh;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, ssh: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaSSH
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
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Enable the use of SSH/SCP for downloading demos from a remote server.</SubHeading>
                        <form.AppField
                            name={'enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable SSH downloader'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>SSH username</SubHeading>
                        <form.AppField
                            name={'username'}
                            children={(field) => {
                                return <field.TextField label={'SSH username'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            SSH port to use. This assumes all servers are configured using the same port.
                        </SubHeading>
                        <form.AppField
                            name={'port'}
                            children={(field) => {
                                return <field.TextField label={'SSH port'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Path to your private key if using key based authentication.</SubHeading>
                        <form.AppField
                            name={'private_key_path'}
                            children={(field) => {
                                return <field.TextField label={'Path to private key'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Password when using standard auth. Passphrase to unlock the private key when using key auth.
                        </SubHeading>
                        <form.AppField
                            name={'password'}
                            children={(field) => {
                                return <field.TextField label={'SSH/Private key password'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>How often to connect to remove systems and check for demos.</SubHeading>
                        <form.AppField
                            name={'update_interval'}
                            children={(field) => {
                                return <field.TextField label={'Check frequency (seconds)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>Connection timeout.</SubHeading>
                        <form.AppField
                            name={'timeout'}
                            children={(field) => {
                                return <field.TextField label={'Connection timeout (seconds)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Format for generating a path to look for demos. Use <kbd>%s</kbd> as a substitution for the
                            short server name.
                        </SubHeading>
                        <form.AppField
                            name={'demo_path_fmt'}
                            children={(field) => {
                                return <field.TextField label={'Path format for retrieving demos'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Format for generating a path to look for stac anticheat logs. Use <kbd>%s</kbd> as a
                            substitution for the short server name.
                        </SubHeading>
                        <form.AppField
                            name={'stac_path_fmt'}
                            children={(field) => {
                                return <field.TextField label={'Path format for retrieving stac logs'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};

const ExportsSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const defaultValues: z.input<typeof schemaExports> = settings.exports;
    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutate({ ...settings, exports: value });
        },
        defaultValues,
        validators: {
            onSubmit: schemaExports
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
                    await form.handleSubmit();
                }}
            >
                <ConfigContainer>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Comma separated list of authorized keys which can access these resources. If no keys are
                            specified, access will be granted to everyone. Append key to query with{' '}
                            <kbd>&key=value</kbd>
                        </SubHeading>
                        <form.AppField
                            name={'authorized_keys'}
                            children={(field) => {
                                return <field.TextField label={'Authorized Keys (comma separated).'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable exporting of a TF2 Bot Detector compatible player list. Only exports users banned
                            with the cheater reason.
                        </SubHeading>
                        <form.AppField
                            name={'bd_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable tf2 bot detector compatible export'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <SubHeading>
                            Enable exporting of a SRCDS banned_user.cfg compatible player list. Only exports users
                            banned with the cheater reason.
                        </SubHeading>
                        <form.AppField
                            name={'valve_enabled'}
                            children={(field) => {
                                return <field.CheckboxField label={'Enable srcds formatted ban list'} />;
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
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </ConfigContainer>
            </form>
        </TabSection>
    );
};
