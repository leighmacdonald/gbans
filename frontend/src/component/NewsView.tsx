import React, { useEffect } from 'react';
import { useState } from 'react';
import { apiGetNewsLatest, NewsEntry } from '../api/news';
import Stack from '@mui/material/Stack';
import { marked } from 'marked';
import Paper from '@mui/material/Paper';
import { Pagination } from '@mui/material';
import { Heading } from './Heading';
export interface NewsViewProps {
    itemsPerPage: number;
}
export const NewsView = ({ itemsPerPage }: NewsViewProps) => {
    const [articles, setArticles] = useState<NewsEntry[]>();
    const [page, setPage] = useState<number>(0);

    useEffect(() => {
        apiGetNewsLatest().then((latest) => {
            const art = latest || [];
            setArticles(art);
        });
    }, [itemsPerPage]);

    return (
        <Stack spacing={3}>
            {articles
                ?.slice(page * itemsPerPage, page * itemsPerPage + itemsPerPage)
                .map((article) => {
                    return (
                        <Paper
                            elevation={1}
                            key={`news_` + article.news_id}
                            sx={{ padding: 3 }}
                        >
                            <Heading>{article.title}</Heading>
                            <div
                                className={'content'}
                                dangerouslySetInnerHTML={{
                                    __html: marked.parse(article.body_md)
                                }}
                            />
                        </Paper>
                    );
                })}
            <Pagination
                count={articles ? Math.ceil(articles.length / itemsPerPage) : 0}
                defaultValue={1}
                onChange={(_, newPage) => {
                    setPage(newPage - 1);
                }}
            ></Pagination>
        </Stack>
    );
};
