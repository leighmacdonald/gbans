import * as yup from 'yup';
import { PermissionLevel } from '../../api';

export const minStringValidator = (name: string, minimum = 1) =>
    yup.string().min(minimum).label(`${name} of the contest`).required(`${name} is required`);

export const minNumberValidator = (name: string, minimum = 1) =>
    yup.number().min(minimum).label(`Minimum ${name}`).required(`${name} is required`);

export const numberValidator = (name: string) => yup.number().label(name).required(`${name} is required`);

export const dateDefinedValidator = (name = 'Date') => yup.date().required(`${name} is required`);

export const mimeTypesValidator = () => {
    return yup
        .string()
        .label('Allowed mimetypes (none = all allowed)')
        .test(
            'valid-mime-format',
            'Invalid mimetype format, must be comma separated list eg: application/gzip,image/gif  (no spaces allowed)',
            (value) => {
                if (!value) {
                    return true;
                }

                const parts = value?.split(',');
                return parts?.filter((p) => p.match(/^\S+\/\S+$/)).length == parts?.length;
            }
        );
};

export const dateAfterValidator = (key: string, name = 'Date') =>
    dateDefinedValidator(name).when(key, (value, schema) =>
        !value ? schema : yup.date().min(value, `${name} must come after first date`)
    );

export const boolDefinedValidator = (name: string) =>
    yup.boolean().defined().label(`${name} of the contest`).required(`${name} is required`);

export const permissionValidator = (minimum: PermissionLevel = PermissionLevel.User, label = 'Min Permissions') => {
    return yup
        .number()
        .oneOf([PermissionLevel.User, PermissionLevel.Moderator, PermissionLevel.Admin])
        .min(minimum)
        .label(label)
        .required('Minimum permission value required');
};
