import { SyntheticEvent, useCallback, useState } from 'react';
import PublishedWithChangesIcon from '@mui/icons-material/PublishedWithChanges';
import SaveIcon from '@mui/icons-material/Save';
import UnpublishedIcon from '@mui/icons-material/Unpublished';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { apiNewsSave, NewsEntry } from '../api/news';
import { MarkDownRenderer } from '../component/MarkdownRenderer';
import { NewsList } from '../component/NewsList';
import { TabPanel } from '../component/TabPanel';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const AdminNewsPage = () => {
    const [setTabValue, setTabSetTabValue] = useState(0);
    const { sendFlash } = useUserFlashCtx();
    const handleChange = (_: SyntheticEvent, newValue: number) => {
        setTabSetTabValue(newValue);
    };
    const [selectedNewsEntry, setSelectedNewsEntry] = useState<NewsEntry>({
        news_id: 0,
        body_md: '',
        is_published: false,
        title: '',
        created_on: new Date(),
        updated_on: new Date()
    });

    const onSave = useCallback(() => {
        apiNewsSave(selectedNewsEntry)
            .then((response) => {
                setSelectedNewsEntry(response);
                sendFlash(
                    'success',
                    `News published successfully: ${selectedNewsEntry.title}`
                );
            })
            .catch((e) => {
                sendFlash('error', 'Failed to save news');
                logErr(e);
            });
    }, [selectedNewsEntry, sendFlash]);

    return (
        <Grid container spacing={2}>
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
                            <MarkDownRenderer
                                body_md={selectedNewsEntry.body_md}
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

export default AdminNewsPage;
