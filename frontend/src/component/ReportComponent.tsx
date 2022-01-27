import React, { useCallback, useEffect, useState } from 'react';
import {
    apiCreateReportMessage,
    apiGetLogs,
    apiGetReportMessages,
    Person,
    Report,
    ReportMessage,
    ReportMessagesResponse,
    UserMessageLog
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
            {value === index && (
                <Box sx={{ p: 3 }}>
                    <Typography>{children}</Typography>
                </Box>
            )}
        </div>
    );
}

export interface UserMessageViewProps {
    author: Person;
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
                title={author.personaname}
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
        const submit = async () => {
            const resp = await apiCreateReportMessage(
                report.report_id,
                comment
            );
            setMessages([
                ...messages,
                { author: currentUser.player, message: resp }
            ]);
            setComment('');
        };
        submit();
    }, [comment, messages, report.report_id]);

    useEffect(() => {
        const loadMessages = async () => {
            const resp = await apiGetReportMessages(report.report_id);
            setMessages(resp);
        };
        loadMessages();

        const loadUserLogs = async () => {
            const resp = await apiGetLogs(report.reported_id, 100);
            setLogs(resp);
        };
        loadUserLogs();
    }, [report]);

    return (
        <Grid container>
            <Grid item xs={12}>
                <Box padding={2}>
                    <Typography variant={'h2'}>{report?.title}</Typography>
                </Box>
                <Paper elevation={1} sx={{ width: '100%' }}>
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
                            <Tab label="Chat Logs" />
                            <Tab label="Connections" />
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
                        <ImageList variant="masonry" cols={3} gap={8}>
                            {report.media_ids.map((item_id) => (
                                <ImageListItem key={item_id}>
                                    <img
                                        src={`/api/download/report/${item_id}`}
                                        alt={''}
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
                <Stack padding={2} spacing={2}>
                    {messages &&
                        messages.map((m) => (
                            <UserMessageView
                                author={m.author}
                                message={m.message}
                                key={m.message.report_message_id}
                            />
                        ))}
                </Stack>
                <Paper elevation={1}>
                    <Paper elevation={1} sx={{ marginTop: '1rem' }}>
                        <TextField
                            label="Comment"
                            id="comment"
                            minRows={10}
                            variant={'filled'}
                            margin={'normal'}
                            multiline
                            fullWidth
                            value={comment}
                            onChange={(v) => {
                                setComment(v.target.value);
                            }}
                        />
                        <Button
                            onClick={onSubmitMessage}
                            fullWidth
                            variant={'contained'}
                            color={'primary'}
                            endIcon={<SendIcon />}
                        >
                            Send Comment
                        </Button>
                    </Paper>
                </Paper>
            </Grid>
        </Grid>
    );
};
