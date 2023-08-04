import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import { NewsList } from '../component/NewsList';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import PublishedWithChangesIcon from '@mui/icons-material/PublishedWithChanges';
import UnpublishedIcon from '@mui/icons-material/Unpublished';
import ButtonGroup from '@mui/material/ButtonGroup';
import SaveIcon from '@mui/icons-material/Save';
import { apiNewsSave, NewsEntry } from '../api/news';
import { marked } from 'marked';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { TabPanel } from '../component/TabPanel';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminNews = () => {
    const [setTabValue, setTabSetTabValue] = React.useState(0);
    const { sendFlash } = useUserFlashCtx();
    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setTabSetTabValue(newValue);
    };
    const [bodyHTML, setBodyHTML] = React.useState('');
    const [selectedNewsEntry, setSelectedNewsEntry] = useState<NewsEntry>({
        news_id: 0,
        body_md: '',
        is_published: false,
        title: ''
    });
    const onSave = useCallback(() => {
        apiNewsSave(selectedNewsEntry)
            .then((response) => {
                if (!response.status || !response.result) {
                    sendFlash('error', 'Failed to save news');
                    return;
                }
                setSelectedNewsEntry(response.result);
                sendFlash(
                    'success',
                    `News published successfully: ${selectedNewsEntry.title}`
                );
            })
            .catch(logErr);
    }, [selectedNewsEntry, sendFlash]);

    useEffect(() => {
        setBodyHTML(marked(selectedNewsEntry.body_md, { mangle: false }));
    }, [selectedNewsEntry.body_md]);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid xs={8}>
                <Paper elevation={1}>
                    <Stack spacing={3} padding={3}>
                        <Box color={'primary'}>
                            <Typography variant={'h4'}>
                                {selectedNewsEntry.news_id > 0
                                    ? 'Edit News Entry'
                                    : 'Create News Entry'}
                            </Typography>
                        </Box>
                        <TextField
                            id="headline"
                            label="Headline"
                            fullWidth
                            value={selectedNewsEntry.title}
                            onChange={(v) => {
                                setSelectedNewsEntry((prevState) => {
                                    return {
                                        ...prevState,
                                        title: v.target.value
                                    };
                                });
                            }}
                        />
                        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                            <Tabs
                                value={setTabValue}
                                onChange={handleChange}
                                aria-label="Markdown & HTML Preview"
                            >
                                <Tab label="Edit" />
                                <Tab label="Preview" />
                            </Tabs>
                        </Box>
                        <TabPanel value={setTabValue} index={0}>
                            <TextField
                                id="body"
                                label="Body (Markdown)"
                                fullWidth
                                multiline
                                minRows={15}
                                value={selectedNewsEntry.body_md}
                                onChange={(event) => {
                                    setSelectedNewsEntry((prevState) => {
                                        return {
                                            ...prevState,
                                            body_md: event.target.value
                                        };
                                    });
                                }}
                            />
                        </TabPanel>
                        <TabPanel value={setTabValue} index={1}>
                            <article
                                dangerouslySetInnerHTML={{ __html: bodyHTML }}
                            />
                        </TabPanel>
                    </Stack>
                </Paper>
            </Grid>
            <Grid xs={4}>
                <Stack spacing={3}>
                    <ButtonGroup fullWidth>
                        <Button
                            variant="contained"
                            endIcon={<UnpublishedIcon />}
                            color={'error'}
                            disabled={!selectedNewsEntry.is_published}
                            onClick={() => {
                                setSelectedNewsEntry((prevState) => {
                                    return {
                                        ...prevState,
                                        is_published: false
                                    };
                                });
                            }}
                        >
                            UnPublish
                        </Button>
                        <Button
                            variant="contained"
                            endIcon={<PublishedWithChangesIcon />}
                            color={'success'}
                            disabled={selectedNewsEntry.is_published}
                            onClick={() => {
                                setSelectedNewsEntry((prevState) => {
                                    return {
                                        ...prevState,
                                        is_published: true
                                    };
                                });
                            }}
                        >
                            Publish
                        </Button>
                    </ButtonGroup>
                    <ButtonGroup fullWidth>
                        <Button
                            variant="contained"
                            endIcon={<SaveIcon />}
                            color={'success'}
                            onClick={onSave}
                        >
                            {selectedNewsEntry.news_id > 0
                                ? 'Save Article'
                                : 'Create Article'}
                        </Button>
                    </ButtonGroup>

                    <Paper elevation={1}>
                        <NewsList setSelectedNewsEntry={setSelectedNewsEntry} />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
