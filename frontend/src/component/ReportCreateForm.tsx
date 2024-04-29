import { JSX, useEffect } from 'react';
import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import { sessionKeyDemoName, sessionKeyReportPersonMessageIdName, sessionKeyReportSteamID } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';

// interface ReportValues {
//     steam_id: string;
//     body_md: string;
//     reason: BanReason;
//     reason_text: string;
//     demo_name: string;
//     demo_tick?: number;
//     person_message_id: number;
// }

// const validationSchema = yup.object({
//     steam_id: steamIdValidator(),
//     body_md: yup.string().min(10, 'Message too short (min 10)').required('Description is required'),
//     reason: banReasonFieldValidator,
//     reason_text: banReasonTextFieldValidator,
//     //person_message_id: yup.number().min(1, 'Invalid message id').optional()
//     demo_name: yup.string().optional(),
//     demo_tick: yup.number().min(0, 'invalid demo tick value').optional()
// });

export const ReportCreateForm = (): JSX.Element => {
    // const [personMessageID] = useState(parseInt(sessionStorage.getItem(sessionKeyReportPersonMessageIdName) ?? '0', 10));
    // const [demoName] = useState(sessionStorage.getItem(sessionKeyDemoName) ?? '');
    //
    // const navigate = useNavigate();

    useEffect(() => {
        return () => {
            sessionStorage.removeItem(sessionKeyDemoName);
            sessionStorage.removeItem(sessionKeyReportPersonMessageIdName);
            sessionStorage.removeItem(sessionKeyReportSteamID);
        };
    }, []);

    // const onSubmit = useCallback(
    //     async (values: ReportValues, formikHelpers: FormikHelpers<ReportValues>) => {
    //         try {
    //             const report = await apiCreateReport({
    //                 demo_name: values.demo_name,
    //                 demo_tick: values.demo_tick ?? 0,
    //                 description: values.body_md,
    //                 reason_text: values.reason_text,
    //                 target_id: values.steam_id,
    //                 person_message_id: values.person_message_id,
    //                 reason: values.reason
    //             });
    //             await navigate({ to: `/report/${report.report_id}` });
    //             formikHelpers.resetForm();
    //         } catch (e) {
    //             logErr(e);
    //         }
    //     },
    //     [navigate]
    // );

    return (
        <ContainerWithHeader title={'Create a New Report'} iconLeft={<EditNotificationsIcon />} spacing={2}>
            {/*<Formik<ReportValues>*/}
            {/*    onSubmit={onSubmit}*/}
            {/*    validateOnBlur={true}*/}
            {/*    validationSchema={validationSchema}*/}
            {/*    initialValues={{*/}
            {/*        demo_name: demoName,*/}
            {/*        demo_tick: undefined,*/}
            {/*        person_message_id: personMessageID,*/}
            {/*        body_md: '',*/}
            {/*        reason: BanReason.Cheating,*/}
            {/*        reason_text: '',*/}
            {/*        steam_id: ''*/}
            {/*    }}*/}
            {/*>*/}
            <Stack spacing={1}>
                {/*<ProfileSelectionField />*/}
                {/*<BanReasonField />*/}
                {/*<BanReasonTextField paired />*/}
                {/*<Stack direction={'row'} spacing={1}>*/}
                {/*    <DemoNameField />*/}
                {/*    <DemTickField />*/}
                {/*</Stack>*/}

                {/*{personMessageID != undefined && personMessageID > 0 && (*/}
                {/*    <PlayerMessageContext playerMessageId={personMessageID} padding={5} />*/}
                {/*)}*/}
                {/*<Box minHeight={365}>*/}
                {/*    <MDBodyField />*/}
                {/*</Box>*/}
                <ButtonGroup>
                    {/*<ResetButton />*/}
                    {/*<SubmitButton label={'Report'} startIcon={<SendIcon />} />*/}
                </ButtonGroup>
            </Stack>
            {/*</Formik>*/}
        </ContainerWithHeader>
    );
};
