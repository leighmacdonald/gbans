import { PropsWithChildren, ReactNode, useState } from 'react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import BugReportIcon from '@mui/icons-material/BugReport';
import EmergencyRecordingIcon from '@mui/icons-material/EmergencyRecording';
import GradingIcon from '@mui/icons-material/Grading';
import HeadsetMicIcon from '@mui/icons-material/HeadsetMic';
import LanIcon from '@mui/icons-material/Lan';
import SettingsIcon from '@mui/icons-material/Settings';
import ShareIcon from '@mui/icons-material/Share';
import TravelExploreIcon from '@mui/icons-material/TravelExplore';
import TroubleshootIcon from '@mui/icons-material/Troubleshoot';
import WebAssetIcon from '@mui/icons-material/WebAsset';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetSettings, apiSaveSettings, Config } from '../api/admin.ts';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { commonTableSearchSchema } from '../util/table.ts';

const serversSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['server_id', 'short_name', 'name', 'address', 'port', 'region', 'cc', 'enable_stats', 'is_enabled'])
        .optional()
});

export const Route = createFileRoute('/_admin/admin/settings')({
    component: AdminServers,
    validateSearch: (search) => serversSearchSchema.parse(search),
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
    | 'demos'
    | 'patreon'
    | 'discord'
    | 'logging'
    | 'sentry'
    | 'ip2location'
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
            <Typography variant={'subtitle1'}>{description}</Typography>
            {children}
        </Grid>
    );
};

function AdminServers() {
    const { sendFlash } = useUserFlashCtx();
    const settings = Route.useLoaderData();
    const [tab, setTab] = useState<tabs>('general');

    const mutation = useMutation({
        mutationKey: ['adminSettings'],
        mutationFn: async (variables: Config) => {
            return await apiSaveSettings(variables);
        },
        onSuccess: () => {
            sendFlash('success', 'Settings updates successfully');
        },
        onError: (error) => {
            sendFlash('error', `Error updating settings: ${error}`);
        }
    });

    const onTabClick = (newTab: tabs) => {
        setTab(newTab);
        console.log(`set tab ${newTab}`);
    };

    return (
        <>
            <Title>Edit Settings</Title>

            <ContainerWithHeaderAndButtons title={'System Settings'} iconLeft={<SettingsIcon />}>
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
                                tab={'demos'}
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
                                tab={'ip2location'}
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
                                tab={'exports'}
                                onClick={onTabClick}
                                icon={<ShareIcon />}
                                currentTab={tab}
                                label={'Exports'}
                            />
                        </Stack>
                    </Grid>
                    <GeneralSection tab={tab} settings={settings} mutate={mutation.mutate} />
                    <TabSection
                        tab={'filters'}
                        currentTab={tab}
                        label={'Word Filters'}
                        description={
                            'Word filters are a form of auto-moderation that scans' +
                            'incoming chat logs for matching values and handles them accordingly'
                        }
                    >
                        filters
                    </TabSection>
                </Grid>
            </ContainerWithHeaderAndButtons>
        </>
    );
}

const GeneralSection = ({ tab, settings, mutate }: { tab: tabs; settings: Config; mutate: (s: Config) => void }) => {
    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            console.log(value);
            mutate({ ...settings, general: value });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            srcds_log_addr: settings.general.srcds_log_addr,
            file_serve_mode: settings.general.file_serve_mode,
            steam_key: settings.general.steam_key,
            mode: settings.general.mode,
            site_name: settings.general.site_name
        }
    });

    return (
        <TabSection
            tab={'general'}
            currentTab={tab}
            label={'General'}
            description={'Core settings that dont belong to a subcategory'}
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
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} fullwidth={true} />;
                            }}
                        />
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'srcds_log_addr'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} fullwidth={true} />;
                            }}
                        />
                    </Grid>
                    <Grid xs={12}>
                        <Field
                            name={'steam_key'}
                            validators={{
                                onChange: z.string()
                            }}
                            children={(props) => {
                                return <TextFieldSimple {...props} fullwidth={true} />;
                            }}
                        />
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
