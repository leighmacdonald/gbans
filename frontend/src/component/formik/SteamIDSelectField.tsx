import { useFormikContext } from 'formik';
import { PersonCell } from '../PersonCell';

export const SteamIDSelectField = ({
    steam_id,
    field_name,
    avatarhash,
    personaname
}: {
    steam_id: string;
    field_name: string;
    avatarhash: string;
    personaname?: string;
}) => {
    const { setFieldValue, submitForm } = useFormikContext<{
        source_id: string;
    }>();

    return (
        <PersonCell
            onClick={async () => {
                await setFieldValue(field_name, steam_id);
                await submitForm();
            }}
            steam_id={steam_id}
            personaname={personaname ?? ''}
            avatar_hash={avatarhash}
        />
    );
};
