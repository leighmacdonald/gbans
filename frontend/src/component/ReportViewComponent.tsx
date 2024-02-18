import { useCallback, useState, JSX, useMemo, SyntheticEvent } from 'react';
import DescriptionIcon from '@mui/icons-material/Description';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import LanIcon from '@mui/icons-material/Lan';
import MessageIcon from '@mui/icons-material/Message';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import TabContext from '@mui/lab/TabContext';
import TabList from '@mui/lab/TabList';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import {
    apiCreateReportMessage,
    apiDeleteReportMessage,
    PermissionLevel,
    Report,
    ReportMessage
} from '../api';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { useReportMessages } from '../hooks/useReportMessages';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors';
import { ContainerWithHeader } from './ContainerWithHeader';
import { MDBodyField } from './MDBodyField';
import { MarkDownRenderer } from './MarkdownRenderer';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ReportMessageView } from './ReportMessageView';
import { SourceBansList } from './SourceBansList';
import { TabPanel } from './TabPanel';
import { ResetButton, SubmitButton } from './modal/Buttons';
import { BanHistoryTable } from './table/BanHistoryTable';
import { ConnectionHistoryTable } from './table/ConnectionHistoryTable';
import { PersonMessageTable } from './table/PersonMessageTable';

interface ReportComponentProps {
    report: Report;
}

interface ReportViewValues {
    body_md: string;
}

export const ReportViewComponent = ({
    report
}: ReportComponentProps): JSX.Element => {
    const theme = useTheme();
    const { data: messagesServer } = useReportMessages(report.report_id);
    const [newMessages, setNewMessages] = useState<ReportMessage[]>([]);
    const [deletedMessages, setDeletedMessages] = useState<number[]>([]);
    const [value, setValue] = useState<number>(0);
    const [banCount, setBanCount] = useState(0);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();

    const messages = useMemo(() => {
        return [...messagesServer, ...newMessages].filter(
            (m) => !deletedMessages.includes(m.report_message_id)
        );
    }, [deletedMessages, messagesServer, newMessages]);

    const handleChange = (_: SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    const onSubmit = useCallback(
        async (
            values: ReportViewValues,
            formikHelpers: FormikHelpers<ReportViewValues>
        ) => {
            try {
                const message = await apiCreateReportMessage(
                    report.report_id,
                    values.body_md
                );
                setNewMessages((prevState) => {
                    return [...prevState, message];
                });
                formikHelpers.resetForm();
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error trying to create message');
            }
        },
        [report.report_id, sendFlash]
    );

    const onDelete = useCallback(
        async (message_id: number) => {
            try {
                await apiDeleteReportMessage(message_id);
                setDeletedMessages((prevState) => {
                    return [...prevState, message_id];
                });
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Failed to delete message');
            }
        },
        [sendFlash]
    );

    return (
        <Grid container>
            <Grid xs={12}>
                <TabContext value={value}>
                    <Stack spacing={2}>
                        <ContainerWithHeader
                            title={'Report Overview'}
                            iconLeft={<ReportIcon />}
                        >
                            <Box
                                sx={{
                                    borderBottom: 1,
                                    borderColor: 'divider',
                                    backgroundColor:
                                        theme.palette.background.paper
                                }}
                            >
                                <TabList
                                    variant={'fullWidth'}
                                    onChange={handleChange}
                                    aria-label="ReportCreatePage detail tabs"
                                >
                                    <Tab
                                        label="Description"
                                        icon={<DescriptionIcon />}
                                        iconPosition={'start'}
                                    />
                                    {currentUser.permission_level >=
                                        PermissionLevel.Moderator && (
                                        <Tab
                                            sx={{ height: 20 }}
                                            label={`Chat Logs`}
                                            icon={<MessageIcon />}
                                            iconPosition={'start'}
                                        />
                                    )}
                                    {currentUser.permission_level >=
                                        PermissionLevel.Moderator && (
                                        <Tab
                                            label={`Connections`}
                                            icon={<LanIcon />}
                                            iconPosition={'start'}
                                        />
                                    )}
                                    {currentUser.permission_level >=
                                        PermissionLevel.Moderator && (
                                        <Tab
                                            label={`Ban History ${banCount ? `(${banCount})` : ''}`}
                                            icon={<ReportGmailerrorredIcon />}
                                            iconPosition={'start'}
                                        />
                                    )}
                                </TabList>
                            </Box>

                            <TabPanel value={value} index={0}>
                                {report && (
                                    <Box minHeight={300}>
                                        <MarkDownRenderer
                                            body_md={report.description}
                                        />
                                    </Box>
                                )}
                            </TabPanel>

                            <TabPanel value={value} index={1}>
                                <Box minHeight={300}>
                                    <PersonMessageTable
                                        steam_id={report.target_id}
                                    />
                                </Box>
                            </TabPanel>
                            <TabPanel value={value} index={2}>
                                <Box minHeight={300}>
                                    <ConnectionHistoryTable
                                        steam_id={report.target_id}
                                    />
                                </Box>
                            </TabPanel>
                            <TabPanel value={value} index={3}>
                                <Box
                                    minHeight={300}
                                    style={{
                                        display: value == 3 ? 'block' : 'none'
                                    }}
                                >
                                    <BanHistoryTable
                                        steam_id={report.target_id}
                                        setBanCount={setBanCount}
                                    />
                                </Box>
                            </TabPanel>
                        </ContainerWithHeader>
                        {report.demo_name != '' && (
                            <Paper>
                                <Stack direction={'row'} padding={1}>
                                    <Typography
                                        padding={2}
                                        variant={'button'}
                                        alignContent={'center'}
                                    >
                                        Demo&nbsp;Info
                                    </Typography>
                                    <Typography
                                        padding={2}
                                        variant={'body1'}
                                        alignContent={'center'}
                                    >
                                        Tick:&nbsp;{report.demo_tick}
                                    </Typography>
                                    <Button
                                        fullWidth
                                        startIcon={<FileDownloadIcon />}
                                        component={Link}
                                        variant={'text'}
                                        href={`${window.gbans.asset_url}/${window.gbans.bucket_demo}/${report.demo_name}`}
                                        color={'primary'}
                                    >
                                        {report.demo_name}
                                    </Button>
                                </Stack>
                            </Paper>
                        )}

                        {report.person_message_id > 0 && (
                            <ContainerWithHeader title={'Message Context'}>
                                <PlayerMessageContext
                                    playerMessageId={report.person_message_id}
                                    padding={4}
                                />
                            </ContainerWithHeader>
                        )}

                        {currentUser.permission_level >=
                            PermissionLevel.Moderator && (
                            <SourceBansList
                                steam_id={report.source_id}
                                is_reporter={true}
                            />
                        )}

                        {currentUser.permission_level >=
                            PermissionLevel.Moderator && (
                            <SourceBansList
                                steam_id={report.target_id}
                                is_reporter={false}
                            />
                        )}

                        {messages.map((m) => (
                            <ReportMessageView
                                onDelete={onDelete}
                                message={m}
                                key={`report-msg-${m.report_message_id}`}
                            />
                        ))}
                        <Paper elevation={1}>
                            <Formik<ReportViewValues>
                                initialValues={{ body_md: '' }}
                                onSubmit={onSubmit}
                            >
                                <Stack spacing={2} padding={1}>
                                    <Box minHeight={465}>
                                        <MDBodyField />
                                    </Box>
                                    <ButtonGroup>
                                        <ResetButton />
                                        <SubmitButton />
                                    </ButtonGroup>
                                </Stack>
                            </Formik>
                        </Paper>
                    </Stack>
                </TabContext>
            </Grid>
        </Grid>
    );
};
