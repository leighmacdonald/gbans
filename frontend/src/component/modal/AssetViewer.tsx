import { useMemo } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PermMediaIcon from '@mui/icons-material/PermMedia';
import { CardActionArea, CardMedia, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import { Player } from 'video-react';
import 'video-react/dist/video-react.css';
import { assetURL } from '../../api/media';
import { Asset, mediaType, MediaTypes } from '../../schema/asset';
import { Heading } from '../Heading';

const VideoAsset = ({ asset }: AssetViewerProps) => (
    <Box>
        <Player>
            <source src={assetURL(asset)} type={asset.mime_type} />
        </Player>
    </Box>
);

const ImageAsset = ({ asset }: AssetViewerProps) => {
    return (
        <Card>
            <CardActionArea>
                <CardMedia
                    component="img"
                    //height="140"
                    image={assetURL(asset)}
                    alt={asset.name}
                />
            </CardActionArea>
        </Card>
    );
};

interface AssetViewerProps {
    asset: Asset;
}

export const AssetViewer = NiceModal.create((asset: Asset) => {
    const modal = useModal();

    const content = useMemo(() => {
        switch (mediaType(asset.mime_type)) {
            case MediaTypes.image:
                return <ImageAsset asset={asset} />;
            case MediaTypes.video:
                return <VideoAsset asset={asset} />;
            default:
                return <></>;
        }
    }, [asset]);

    return (
        <Dialog fullWidth {...muiDialogV5(modal)} fullScreen={mediaType(asset.mime_type) == MediaTypes.image}>
            <DialogTitle component={Heading} iconLeft={<PermMediaIcon />}>
                {`${asset.name}`}
            </DialogTitle>

            <DialogContent>{content}</DialogContent>

            <DialogActions>{/*<CloseButton onClick={modal.hide} />*/}</DialogActions>
        </Dialog>
    );
});
