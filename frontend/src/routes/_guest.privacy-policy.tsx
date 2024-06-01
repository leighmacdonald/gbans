import { PropsWithChildren } from 'react';
import PolicyIcon from '@mui/icons-material/Policy';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { Title } from '../component/Title.tsx';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';

export const Route = createFileRoute('/_guest/privacy-policy')({
    component: PrivacyPolicy
});

const PPBox = ({ heading, children }: { heading: string } & PropsWithChildren) => {
    return (
        <Grid md={6} xs={12} padding={2}>
            <Typography variant={'h3'} paddingBottom={3} paddingTop={2} sx={{ textTransform: 'capitalize' }}>
                {heading}
            </Typography>
            {children}
        </Grid>
    );
};

function PrivacyPolicy() {
    const { appInfo } = useAppInfoCtx();

    return (
        <>
            <Title>Privacy Policy</Title>

            <ContainerWithHeader title={'Privacy Policy'} padding={2} iconLeft={<PolicyIcon />}>
                <Grid container>
                    <PPBox heading={'Why We Collect Personal Information'}>
                        <ul>
                            <li>
                                Where it is necessary for the purposes of the legitimate and legal interests of{' '}
                                {appInfo.site_name}, except where such interests are overridden by your prevailing
                                legitimate interests and rights; or
                            </li>
                            <li>Where you have given consent to it.</li>
                        </ul>
                    </PPBox>
                    <PPBox heading={'What Types Of Personal Information We Collect'}>
                        <ul>
                            <li>
                                <Typography variant={'body1'}>Username</Typography>
                            </li>

                            <li>
                                <Typography variant={'body1'}>Steam ID</Typography>
                            </li>

                            <li>
                                <Typography variant={'body1'}>IP Address</Typography>
                            </li>

                            <li>
                                <Typography variant={'body1'}>Steam Avatar</Typography>
                            </li>

                            <li>
                                <Typography variant={'body1'}>Chat History</Typography>
                            </li>
                        </ul>
                        <Typography variant={'body1'}>
                            Additionally we may use data from the Steam API according to the{' '}
                            <Link href={'https://store.steampowered.com/subscriber_agreement/'}>
                                Steam Subscriber Agreement
                            </Link>
                        </Typography>
                    </PPBox>
                    <PPBox heading={'How Your Information Is Being Used'}>
                        <ul>
                            <li>
                                <Typography variant={'body1'}>
                                    We collect certain data that is required for the purposes of tracking cheating,
                                    harassment or other violations of Steam and/or Discord policies.
                                </Typography>
                            </li>
                            <li>
                                <Typography variant={'body1'}>
                                    If the data indicates that a Violation has occurred, we will further store the data
                                    for the applicable statute of limitations or until a legal case related to it has
                                    been resolved. Please note that the specific data stored for this purpose may not be
                                    disclosed to you if the disclosure will compromise the mechanism through which we
                                    detect, investigate and prevent such Violations.
                                </Typography>
                            </li>
                        </ul>
                    </PPBox>
                    <PPBox heading={'Cookies'}>
                        <Typography variant={'body1'}>
                            We use cookies on this site. A cookie is a piece of data stored on a site visitors computer
                            to help us improve your access to our site and identify repeat visitors to our site. For
                            instance, when we use a cookie to identify you, you would not have to log in a password more
                            than once, thereby saving time while on our site. Cookies can also enable us to track and
                            target the interests of our users to enhance the experience on our site. Usage of a cookie
                            is in no way linked to any personally identifiable information on our site.
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Payments'}>
                        <Typography variant={'body1'}>
                            Payments are handled via a 3rd party Patreon. See their{' '}
                            <Link href={'https://www.patreon.com/policy'}>policy</Link> for more information.
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Who Has Access To Data'}>
                        <ul>
                            <li>
                                <Typography variant={'body1'}>
                                    Information related to users accounts may be shared to approved 3rd parties for use
                                    in their own respective investigations and safety tools. This information is
                                    restricted to the Steam ID and general reason for ban of the associated account, if
                                    any exists.
                                </Typography>
                            </li>
                            <li>
                                <Typography variant={'body1'}>Contributing content to the site in any way.</Typography>
                            </li>
                        </ul>
                    </PPBox>
                    <PPBox heading={'Keeping your data secure'}>
                        <Typography variant={'body1'}>
                            We are committed to ensuring that any information you provide to us is secure. In order to
                            prevent unauthorized access or disclosure, we have put in place suitable measures and
                            procedures to safeguard and secure the information that we collect.
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Account Deletions'}>
                        <Typography variant={'body1'}>
                            If you would like to delete your account, please contact an admin through the account you
                            wish to delete including reason for deletion. If you do not provide a reason for account
                            deletion, your request will not be granted. We make no guarantee that your request will be
                            granted even if a reason is provided. Per GDPR Article 6(1)(f), {appInfo.site_name} is
                            lawfully permitted to dismiss account deletion requests to protect the legitimate interests
                            of our community by thwarting ban evasions and/or circumvention of our permissions
                            system(s).
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Data Usage and Retention'}>
                        <Typography variant={'body1'}>
                            We will retain your Personal Information only for as long as is necessary for the purposes
                            set out in this Privacy Policy or as required by law. We will use your Personal Information
                            to the extent necessary to comply with our legal obligations, resolve disputes, and enforce
                            our legal agreements and policies.
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Acceptance of this policy'}>
                        <Typography variant={'body1'}>
                            Continued use of our site and services signifies your acceptance of this policy. If you do
                            not accept the policy then please do not use this site. When registering we will further
                            request your explicit acceptance of the privacy policy.
                        </Typography>
                    </PPBox>
                    <PPBox heading={'Changes to this policy'}>
                        <Typography variant={'body1'}>
                            We may make changes to this policy at any time. You may be asked to review and re-accept the
                            information in this policy if it changes in the future.
                        </Typography>
                    </PPBox>
                </Grid>
            </ContainerWithHeader>
        </>
    );
}
