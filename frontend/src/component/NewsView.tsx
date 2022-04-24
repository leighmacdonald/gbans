import React, { useEffect } from 'react';
import { useState } from 'react';
import Typography from '@mui/material/Typography';
import { apiGetNewsLatest, NewsEntry } from '../api/news';
import Stack from '@mui/material/Stack';
import { marked } from 'marked';
import Paper from '@mui/material/Paper';

export const NewsView = () => {
    const [articles, setArticles] = useState<NewsEntry[]>();
    useEffect(() => {
        const f = async () => {
            const latest = await apiGetNewsLatest();
            setArticles((latest as NewsEntry[]) || []);
        };
        f();
    }, []);
    return (
        <Stack spacing={3}>
            {articles?.map((article) => {
                return (
                    <Paper
                        elevation={1}
                        key={`news_` + article.news_id}
                        sx={{ padding: 3 }}
                    >
                        <Typography variant={'h4'}>{article.title}</Typography>
                        <div
                            className={'content'}
                            dangerouslySetInnerHTML={{
                                __html: marked.parse(article.body_md)
                            }}
                        />
                    </Paper>
                );
            })}
        </Stack>
    );
};
