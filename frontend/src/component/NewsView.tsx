import React, { useEffect } from 'react';
import { useState } from 'react';
import Typography from '@mui/material/Typography';
import { apiGetNewsLatest, NewsEntry } from '../api/news';
import Stack from '@mui/material/Stack';
import { marked } from 'marked';

export const NewsView = () => {
    const [articles, setArticles] = useState<NewsEntry[]>();
    const [news, setNews] = useState<NewsEntry>({
        news_id: 0,
        body_md: `
Contrary to popular belief, Lorem Ipsum is not simply random text. It has roots in a piece of classical Latin literature from 45 BC, making it over 2000 years old. Richard McClintock, a Latin professor at Hampden-Sydney College in Virginia, looked up one of the more obscure Latin words, consectetur, from a Lorem Ipsum passage, and going through the cites of the word in classical literature, discovered the undoubtable source. Lorem Ipsum comes from sections 1.10.32 and 1.10.33 of "de Finibus Bonorum et Malorum" (The Extremes of Good and Evil) by Cicero, written in 45 BC. This book is a treatise on the theory of ethics, very popular during the Renaissance. The first line of Lorem Ipsum, "Lorem ipsum dolor sit amet..", comes from a line in section 1.10.32.

The standard chunk of Lorem Ipsum used since the 1500s is reproduced below for those interested. Sections 1.10.32 and 1.10.33 from "de Finibus Bonorum et Malorum" by Cicero are also reproduced in their exact original form, accompanied by English versions from the 1914 translation by H. Rackham.
`,
        is_published: true,
        title: 'Where does it come from?',
        created_on: new Date(),
        updated_on: new Date()
    });
    useEffect(() => {
        const f = async () => {
            const latest = await apiGetNewsLatest();
            if (latest) {
                setArticles(latest as NewsEntry[]);
                setNews(latest[0] as NewsEntry);
            }
        };
        f();
    }, []);
    return (
        <>
            <Typography variant={'h3'}>News {articles?.length}</Typography>
            <Stack>
                <Typography variant={'h4'}>{news.title}</Typography>
                <div
                    className={'content'}
                    dangerouslySetInnerHTML={{
                        __html: marked.parse(news.body_md)
                    }}
                />
            </Stack>
        </>
    );
};
