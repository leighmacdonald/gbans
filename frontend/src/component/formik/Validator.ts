import * as yup from 'yup';
import { PermissionLevel } from '../../api';

export const minStringValidator = (name: string, minimum = 1) =>
    yup
        .string()
        .min(minimum)
        .label(`${name} of the contest`)
        .required(`${name} is required`);

export const minNumberValidator = (name: string, minimum = 1) =>
    yup
        .number()
        .min(minimum)
        .label(`Minimum ${name}`)
        .required(`${name} is required`);

export const dateDefinedValidator = (name = 'Date') =>
    yup.date().required(`${name} is required`);

export const mimeTypesValidator = (minimum: number = 0) => {
    return yup
        .array()
        .min(minimum)
        .label('Allowed mimetypes (none = all allowed)')
        .test('valid-mime-format', 'Invalid mimetype format', (values) => {
            values?.map(
                (mime_type: string) => mime_type.split('/').length == 2
            );
        });
};

export const dateAfterValidator = (key: string, name = 'Date') =>
    dateDefinedValidator(name).when(key, (value, schema) =>
        !value
            ? schema
            : yup.date().min(value, `${name} must come after first date`)
    );

export const boolDefinedValidator = (name: string) =>
    yup
        .boolean()
        .defined()
        .label(`${name} of the contest`)
        .required(`${name} is required`);

export const permissionValidator = (
    minimum: PermissionLevel = PermissionLevel.User,
    label = 'Min Permissions'
) => {
    return yup
        .number()
        .oneOf([
            PermissionLevel.User,
            PermissionLevel.Moderator,
            PermissionLevel.Admin
        ])
        .min(minimum)
        .label(label)
        .required('Minimum permission value required');
};
