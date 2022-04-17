import React, { useEffect, useState } from 'react';
import { ListItem, ListItemIcon, ListItemText } from '@mui/material';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import { apiGetNewsLatest, NewsEntry } from '../api/news';
import FolderIcon from '@mui/icons-material/Folder';

export const NewsList = () => {
    const [news, setNews] = useState<NewsEntry[]>([]);
    useEffect(() => {
        const f = async () => {
            const entries = await apiGetNewsLatest();
            setNews(entries);
        };
        f();
    }, []);
    return (
        <Stack spacing={3} padding={3}>
            <List dense={true}>
                {news.map((n) => {
                    return (
                        <ListItem key={n.news_id}>
                            <ListItemIcon>
                                <FolderIcon />
                            </ListItemIcon>
                            <ListItemText primary={n.title} secondary={null} />
                        </ListItem>
                    );
                })}
            </List>
        </Stack>
    );
};
