import { useEffect, useState } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { apiGetForumOverview, ForumCategory } from '../../api/forum';
import { logErr } from '../../util/errors';

export const serverIDsValidator = yup.number().min(1).label('Forum Category');

interface ServerSelectFieldProps {
    forum_category_id: number;
}

export const ForumCategorySelectField = () => {
    const [categories, setCategories] = useState<ForumCategory[]>();
    const { values, handleChange, touched, errors } =
        useFormikContext<ServerSelectFieldProps>();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetForumOverview(abortController)
            .then((overview) => {
                setCategories(overview.categories);
            })
            .catch(logErr);
        return () => abortController.abort();
    }, []);

    return (
        <FormControl fullWidth>
            <InputLabel id="forum_category_id-label">
                Forum Parent Category
            </InputLabel>
            <Select<number>
                fullWidth
                labelId="forum_category_id-label"
                id="forum_category_id"
                value={values.forum_category_id}
                name={'forum_category_id'}
                label="Forum Parent Category"
                error={
                    touched.forum_category_id &&
                    Boolean(errors.forum_category_id)
                }
                onChange={handleChange}
            >
                {categories &&
                    categories.map((s) => (
                        <MenuItem
                            value={s.forum_category_id}
                            key={`select-cat-${s.forum_category_id}`}
                        >
                            {s.title}
                        </MenuItem>
                    ))}
            </Select>
        </FormControl>
    );
};
