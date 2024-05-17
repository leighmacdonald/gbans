import { SyntheticEvent, useState } from 'react';
import ConstructionIcon from '@mui/icons-material/Construction';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { createFileRoute, useLoaderData } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetPersonSettings, apiSavePersonSettings, PersonSettings } from '../api';
import { Accordion, AccordionDetails, AccordionSummary } from '../component/Accordian.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { Title } from '../component/Title.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { MarkdownField } from '../component/field/MarkdownField.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';

export const Route = createFileRoute('/_auth/settings')({
    component: ProfileSettings,
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
}

const settingsFormSchema = z.object({
    stats_hidden: z.boolean(),
    forum_signature: z.string(),
    forum_profile_messages: z.boolean()
});

function ProfileSettings() {
    const [expanded, setExpanded] = useState<string | false>('general');
    const { sendFlash } = useUserFlashCtx();
    const settings = useLoaderData({ from: '/_auth/settings' }) as PersonSettings;

    const handleChange = (panel: string) => (_: SyntheticEvent, isExpanded: boolean) => {
        setExpanded(isExpanded ? panel : false);
    };

    const mutation = useMutation({
        mutationFn: async (values: SettingsValues) => {
            return await apiSavePersonSettings(
                values.forum_signature,
                values.forum_profile_messages,
                values.stats_hidden
            );
        },
        onSuccess: async () => {
            sendFlash('success', 'Updated Settings');
        },
        onError: (error) => {
            sendFlash('error', `Error Saving ${error}`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: settingsFormSchema
        },
        defaultValues: {
            stats_hidden: settings.stats_hidden,
            forum_signature: settings.forum_signature,
            forum_profile_messages: settings.forum_profile_messages
        }
    });

    return (
        <ContainerWithHeader title={'User Settings'} iconLeft={<ConstructionIcon />}>
            <Title>User Settings</Title>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <>
                    <Accordion expanded={expanded === 'general'} onChange={handleChange('general')}>
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="general-content"
                            id="general-header"
                        >
                            <Typography sx={{ width: '16%', flexShrink: 0 }}>General</Typography>
                            <Typography sx={{ color: 'text.secondary' }}>General account settings</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <Grid xs={12}>
                                    <Field
                                        name={'stats_hidden'}
                                        children={(props) => {
                                            return (
                                                <CheckboxSimple {...props} label={'Hide Profile Stats From Public'} />
                                            );
                                        }}
                                    />
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>
                    <Accordion expanded={expanded === 'forum'} onChange={handleChange('forum')}>
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="forum-content"
                            id="forum-header"
                        >
                            <Typography sx={{ width: '16%', flexShrink: 0 }}>Forum</Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                Configure forum signature and notification
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <Grid xs={12}>
                                    <Field
                                        name={'forum_profile_messages'}
                                        children={(props) => {
                                            return (
                                                <CheckboxSimple {...props} label={'Allow Messages On Public Profile'} />
                                            );
                                        }}
                                    />
                                </Grid>
                                <Grid xs={12}>
                                    <Field
                                        name={'forum_signature'}
                                        children={(props) => {
                                            return <MarkdownField {...props} label={'Forum Signature'} />;
                                        }}
                                    />
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Box>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => (
                                    <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                                )}
                            />
                        </Grid>
                    </Box>
                </>
            </form>
        </ContainerWithHeader>
    );
}
