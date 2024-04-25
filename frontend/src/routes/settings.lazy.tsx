import {
    JSX,
    ReactNode,
    SyntheticEvent,
    useCallback,
    useMemo,
    useState
} from 'react';
import ConstructionIcon from '@mui/icons-material/Construction';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { Formik } from 'formik';
import { apSavePersonSettings, PersonSettings } from '../api';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary
} from '../component/Accordian';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingHeaderIcon } from '../component/LoadingHeaderIcon';
import { MDBodyField } from '../component/MDBodyField';
import { ForumProfileMessagesField } from '../component/formik/ForumProfileMessagesField';
import { StatsHiddenField } from '../component/formik/StatsHiddenField';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';
import { usePersonSettings } from '../hooks/usePersonSettings.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors';

export const Route = createLazyFileRoute('/settings')({
    component: ProfileSettings
});

interface SettingsValues {
    body_md: string;
    forum_profile_messages: boolean;
    stats_hidden: boolean;
}

const SettingRow = ({
    title,
    children
}: {
    title: string;
    children: ReactNode;
}) => {
    return (
        <>
            <Grid xs={2}>
                <Typography sx={{ width: '15%', flexShrink: 0 }}>
                    {title}
                </Typography>
            </Grid>
            <Grid xs={10}>{children}</Grid>
        </>
    );
};

function ProfileSettings() {
    const [expanded, setExpanded] = useState<string | false>('general');
    const { data, loading } = usePersonSettings();
    const [newSettings, setNewSettings] = useState<PersonSettings>();
    const { sendFlash } = useUserFlashCtx();

    const settings = useMemo(() => {
        return newSettings ?? data;
    }, [data, newSettings]);

    const handleChange =
        (panel: string) => (_: SyntheticEvent, isExpanded: boolean) => {
            setExpanded(isExpanded ? panel : false);
        };

    const onSubmit = useCallback(
        async (values: SettingsValues) => {
            try {
                const resp = await apSavePersonSettings(
                    values.body_md,
                    values.forum_profile_messages,
                    values.stats_hidden
                );
                setNewSettings(resp);
                sendFlash('success', 'Updated settings successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error updating settings');
            }
        },
        [sendFlash]
    );

    return (
        <ContainerWithHeader
            title={'User Settings'}
            iconLeft={
                <LoadingHeaderIcon
                    icon={<ConstructionIcon />}
                    loading={loading}
                />
            }
        >
            {!loading && settings && (
                <Formik<SettingsValues>
                    initialValues={{
                        body_md: settings.forum_signature ?? '',
                        forum_profile_messages:
                            settings.forum_profile_messages ?? true,
                        stats_hidden: settings.stats_hidden ?? false
                    }}
                    onSubmit={onSubmit}
                >
                    <>
                        <Accordion
                            expanded={expanded === 'general'}
                            onChange={handleChange('general')}
                        >
                            <AccordionSummary
                                expandIcon={<ExpandMoreIcon />}
                                aria-controls="general-content"
                                id="general-header"
                            >
                                <Typography
                                    sx={{ width: '16%', flexShrink: 0 }}
                                >
                                    General
                                </Typography>
                                <Typography sx={{ color: 'text.secondary' }}>
                                    General account settings
                                </Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Grid container>
                                    <SettingRow title={''}>
                                        <ForumProfileMessagesField />
                                    </SettingRow>
                                    <SettingRow title={''}>
                                        <StatsHiddenField />
                                    </SettingRow>
                                </Grid>
                            </AccordionDetails>
                        </Accordion>
                        <Accordion
                            expanded={expanded === 'forum'}
                            onChange={handleChange('forum')}
                        >
                            <AccordionSummary
                                expandIcon={<ExpandMoreIcon />}
                                aria-controls="forum-content"
                                id="forum-header"
                            >
                                <Typography
                                    sx={{ width: '16%', flexShrink: 0 }}
                                >
                                    Forum
                                </Typography>
                                <Typography sx={{ color: 'text.secondary' }}>
                                    Configure forum signature and notification
                                </Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Grid container>
                                    <SettingRow title={'Signature'}>
                                        <MDBodyField />
                                    </SettingRow>
                                </Grid>
                            </AccordionDetails>
                        </Accordion>

                        <Box>
                            <ButtonGroup>
                                <ResetButton />
                                <SubmitButton label={'Save Settings'} />
                            </ButtonGroup>
                        </Box>
                    </>
                </Formik>
            )}
        </ContainerWithHeader>
    );
}

export default ProfileSettingsLazy;
