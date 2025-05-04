import Box from '@mui/material/Box';

type Props = {
    src: string;
    alt: string;
    height?: number | string;
    width?: number | string;
    maxHeight?: number | string;
    maxWidth?: number | string;
};

export const ImageBox = ({
    src,
    alt,
    maxHeight = undefined,
    maxWidth = undefined,
    width = undefined,
    height = undefined
}: Props) => {
    return (
        <Box
            component="img"
            sx={{
                height: height ? height : undefined,
                width: width ? width : undefined,
                maxHeight: { xs: maxHeight ? maxHeight : undefined },
                maxWidth: { xs: maxWidth ? maxWidth : undefined }
            }}
            alt={alt}
            src={src}
        />
    );
};
