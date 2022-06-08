import React, { useCallback, useEffect, useState } from 'react';
import {
    apiCreateReportMessage,
    apiGetLogs,
    apiGetReportMessages,
    PermissionLevel,
    Report,
    ReportMessage,
    ReportMessagesResponse,
    UserMessageLog,
    UserProfile
} from '../api';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import ImageList from '@mui/material/ImageList';
import ImageListItem from '@mui/material/ImageListItem';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import { formatDistance, parseJSON } from 'date-fns';
import CardContent from '@mui/material/CardContent';
import CardHeader from '@mui/material/CardHeader';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import ButtonGroup from '@mui/material/ButtonGroup';
import { logErr } from '../util/errors';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`tabpanel-${index}`}
            aria-labelledby={`tab-${index}`}
            {...other}
        >
            {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
        </div>
    );
}

export interface UserMessageViewProps {
    author: UserProfile;
    message: ReportMessage;
}

const UserMessageView = ({ author, message }: UserMessageViewProps) => {
    return (
        <Card elevation={1}>
            <CardHeader
                avatar={
                    <Avatar aria-label="Avatar" src={author.avatar}>
                        ?
                    </Avatar>
                }
                action={
                    <IconButton aria-label="Actions">
                        <MoreVertIcon />
                    </IconButton>
                }
                title={author.name}
                subheader={formatDistance(
                    parseJSON(message.created_on),
                    new Date(),
                    { addSuffix: true }
                )}
            />
            <CardContent>
                <Typography variant="body2" color="text.secondary">
                    {message.contents}
                </Typography>
            </CardContent>
        </Card>
    );
};

interface ReportComponentProps {
    report: Report;
}

export const ReportComponent = ({
    report
}: ReportComponentProps): JSX.Element => {
    const [comment, setComment] = useState<string>('');
    const [messages, setMessages] = useState<ReportMessagesResponse[]>([]);
    const [logs, setLogs] = useState<UserMessageLog[]>([]);
    const [value, setValue] = React.useState<number>(0);
    const { currentUser } = useCurrentUserCtx();
    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };
    const onSubmitMessage = useCallback(() => {
        apiCreateReportMessage(report.report_id, comment)
            .then((response) => {
                setMessages([
                    ...messages,
                    { author: currentUser, message: response }
                ]);
                setComment('');
            })
            .catch(logErr);
    }, [comment, messages, report.report_id, currentUser]);

    useEffect(() => {
        apiGetReportMessages(report.report_id)
            .then((r) => {
                setMessages(r);
            })
            .catch(logErr);
    }, [report]);

    useEffect(() => {
        apiGetLogs(`${report.reported_id}`, 100)
            .then((r) => {
                setLogs(r);
            })
            .catch(logErr);
    }, [report]);

    return (
        <Grid container>
            <Grid item xs={12}>
                <Stack spacing={2}>
                    <Paper elevation={1} sx={{ width: '100%', minHeight: 400 }}>
                        <Box
                            sx={{
                                borderBottom: 1,
                                borderColor: 'divider'
                            }}
                        >
                            <Tabs
                                value={value}
                                onChange={handleChange}
                                aria-label="ReportCreatePage detail tabs"
                            >
                                <Tab label="Description" />
                                <Tab label="Evidence" />
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab label="Chat Logs" />
                                )}
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab label="Connections" />
                                )}
                            </Tabs>
                        </Box>

                        <TabPanel value={value} index={0}>
                            {report && (
                                <Typography variant={'body1'}>
                                    {report.description}
                                </Typography>
                            )}
                        </TabPanel>

                        <TabPanel index={value} value={1}>
                            <ImageList
                                variant="masonry"
                                cols={3}
                                gap={8}
                                rowHeight={164}
                            >
                                {(report.media_ids ?? []).map((item_id) => (
                                    <ImageListItem key={item_id}>
                                        <img
                                            style={{
                                                width: '256px',
                                                height: '144px'
                                            }}
                                            src={`/api/download/report/${item_id}`}
                                            alt={'Evidence #' + item_id}
                                            loading="lazy"
                                        />
                                    </ImageListItem>
                                ))}
                            </ImageList>
                        </TabPanel>

                        <TabPanel value={value} index={2}>
                            <Stack>
                                {logs &&
                                    logs.map((log, index) => {
                                        return (
                                            <Box key={index}>
                                                <Typography variant={'body2'}>
                                                    {log.message}
                                                </Typography>
                                            </Box>
                                        );
                                    })}
                            </Stack>
                        </TabPanel>
                        <TabPanel value={value} index={3}>
                            Connection history
                        </TabPanel>
                    </Paper>

                    {messages &&
                        messages.map((m) => (
                            <UserMessageView
                                author={m.author}
                                message={m.message}
                                key={m.message.report_message_id}
                            />
                        ))}
                    <Paper elevation={1}>
                        <Stack spacing={2} padding={1}>
                            <TextField
                                label="Comment"
                                id="comment"
                                minRows={10}
                                variant={'outlined'}
                                margin={'normal'}
                                multiline
                                fullWidth
                                value={comment}
                                onChange={(v) => {
                                    setComment(v.target.value);
                                }}
                            />
                            <ButtonGroup>
                                <Button
                                    onClick={onSubmitMessage}
                                    variant={'contained'}
                                    endIcon={<SendIcon />}
                                    fullWidth={false}
                                >
                                    Send Comment
                                </Button>
                            </ButtonGroup>
                        </Stack>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
