import React, { JSX, ReactNode } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import Link from '@mui/material/Link';

const PPHeading = ({ children }: { children: ReactNode }) => {
    return (
        <Typography variant={'h3'} paddingBottom={3} paddingTop={2}>
            {children}
        </Typography>
    );
};

export const PrivacyPolicyPage = (): JSX.Element => {
    return (
        <ContainerWithHeader title={'Privacy Policy'} padding={2}>
            <Grid xs={12}>
                <PPHeading>Why We Collect Personal Information</PPHeading>

                <ul>
                    <li>
                        Where it is necessary for the purposes of the legitimate
                        and legal interests of {window.gbans.site_name}, except
                        where such interests are overridden by your prevailing
                        legitimate interests and rights; or
                    </li>
                    <li>Where you have given consent to it.</li>
                </ul>
            </Grid>
            <Grid xs={12}>
                <PPHeading>
                    What Types Of Personal Information We Collect
                </PPHeading>

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
            </Grid>
            <Grid xs={12}>
                <Typography variant={'body1'}>
                    Additionally we may use data from the Steam API according to
                    the{' '}
                    <Link
                        href={
                            'https://store.steampowered.com/subscriber_agreement/'
                        }
                    >
                        Steam Subscriber Agreement
                    </Link>
                </Typography>
            </Grid>

            <Grid xs={12}>
                <PPHeading>How Your Information Is Being Used</PPHeading>

                <ul>
                    <li>
                        <Typography variant={'body1'}>
                            We collect certain data that is required for the
                            purposes of tracking cheating, harassment or other
                            violations of Steam and/or Discord policies.
                        </Typography>
                    </li>

                    <li>
                        <Typography variant={'body1'}>
                            If the data indicates that a Violation has occurred,
                            we will further store the data for the applicable
                            statute of limitations or until a legal case related
                            to it has been resolved. Please note that the
                            specific data stored for this purpose may not be
                            disclosed to you if the disclosure will compromise
                            the mechanism through which we detect, investigate
                            and prevent such Violations.
                        </Typography>
                    </li>
                </ul>
            </Grid>

            <Grid xs={12}>
                <PPHeading>Cookies</PPHeading>
                <Typography variant={'body1'}>
                    We use cookies on this site. A cookie is a piece of data
                    stored on a site visitors computer to help us improve your
                    access to our site and identify repeat visitors to our site.
                    For instance, when we use a cookie to identify you, you
                    would not have to log in a password more than once, thereby
                    saving time while on our site. Cookies can also enable us to
                    track and target the interests of our users to enhance the
                    experience on our site. Usage of a cookie is in no way
                    linked to any personally identifiable information on our
                    site.
                </Typography>
            </Grid>

            <Grid xs={12}>
                <PPHeading>Payments</PPHeading>

                <Typography variant={'body1'}>
                    Payments are handled via a 3rd party Patreon. See their{' '}
                    <Link href={'https://www.patreon.com/policy'}>policy</Link>{' '}
                    for more information.
                </Typography>
            </Grid>

            <Grid xs={12}>
                <PPHeading>Who Has Access To Data</PPHeading>

                <ul>
                    <li>
                        <Typography variant={'body1'}>
                            Information related to users accounts may be shared
                            to approved 3rd parties for use in their own
                            respective investigations and safety tools. This
                            information is restricted to the Steam ID and
                            general reason for ban of the associated account, if
                            any exists.
                        </Typography>
                    </li>
                    <li>
                        <Typography variant={'body1'}>
                            Contributing content to the site in any way.
                        </Typography>
                    </li>
                </ul>
            </Grid>

            <Grid xs={12}>
                <PPHeading>Keeping your data secure</PPHeading>

                <Typography variant={'body1'}>
                    We are committed to ensuring that any information you
                    provide to us is secure. In order to prevent unauthorized
                    access or disclosure, we have put in place suitable measures
                    and procedures to safeguard and secure the information that
                    we collect.
                </Typography>
            </Grid>

            <Grid xs={12}>
                <PPHeading>Account Deletions</PPHeading>

                <Typography variant={'body1'}>
                    If you would like to delete your account, please contact an
                    admin through the account you wish to delete including
                    reason for deletion. If you do not provide a reason for
                    account deletion, your request will not be granted. We make
                    no guarantee that your request will be granted even if a
                    reason is provided. Per GDPR Article 6(1)(f),{' '}
                    {window.gbans.site_name} is lawfully permitted to dismiss
                    account deletion requests to protect the legitimate
                    interests of our community by thwarting ban evasions and/or
                    circumvention of our permissions system(s).
                </Typography>
            </Grid>
            <Grid xs={12}>
                <PPHeading>Acceptance of this policy</PPHeading>

                <Typography variant={'body1'}>
                    Continued use of our site and services signifies your
                    acceptance of this policy. If you do not accept the policy
                    then please do not use this site. When registering we will
                    further request your explicit acceptance of the privacy
                    policy.
                </Typography>
            </Grid>
            <Grid xs={12}>
                <PPHeading>Changes to this policy</PPHeading>

                <Typography variant={'body1'}>
                    We may make changes to this policy at any time. You may be
                    asked to review and re-accept the information in this policy
                    if it changes in the future.
                </Typography>
            </Grid>
        </ContainerWithHeader>
    );
};
