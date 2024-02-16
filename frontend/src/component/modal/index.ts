import NiceModal from '@ebay/nice-modal-react';
import loadable from '@loadable/component';

const AssetViewer = loadable(() => import('./AssetViewer'));
const BanASNModal = loadable(() => import('./BanASNModal'));
const BanCIDRModal = loadable(() => import('./BanCIDRModal'));
const BanGroupModal = loadable(() => import('./BanGroupModal'));
const BanSteamModal = loadable(() => import('./BanSteamModal'));
const CIDRBlockEditorModal = loadable(() => import('./CIDRBlockEditorModal'));
const CIDRWhitelistEditorModal = loadable(
    () => import('./CIDRWhitelistEditorModal')
);

const ConfirmationModal = loadable(() => import('./ConfirmationModal'));
const ContestEditor = loadable(() => import('./ContestEditor'));
const ContestEntryDeleteModal = loadable(
    () => import('./ContestEntryDeleteModal')
);
const ContestEntryModal = loadable(() => import('./ContestEntryModal'));
const FileUploadModal = loadable(() => import('./FileUploadModal'));
const FilterEditModal = loadable(() => import('./FilterEditModal'));
const ForumCategoryEditorModal = loadable(
    () => import('./ForumCategoryEditorModal')
);
const ForumForumEditorModal = loadable(() => import('./ForumForumEditorModal'));
const ForumThreadCreatorModal = loadable(
    () => import('./ForumThreadCreatorModal')
);
const ForumThreadEditorModal = loadable(
    () => import('./ForumThreadEditorModal')
);
const MessageContextModal = loadable(() => import('./MessageContextModal'));
const PersonEditModal = loadable(() => import('./PersonEditModal'));
const ServerDeleteModal = loadable(() => import('./ServerDeleteModal'));
const ServerEditorModal = loadable(() => import('./ServerEditorModal'));
const UnbanASNModal = loadable(() => import('./UnbanASNModal'));
const UnbanCIDRModal = loadable(() => import('./UnbanCIDRModal'));
const UnbanGroupModal = loadable(() => import('./UnbanGroupModal'));
const UnbanSteamModal = loadable(() => import('./UnbanSteamModal'));

export const ModalCIDRWhitelistEditor = 'modal-cidr-whitelist-editor';
export const ModalCIDRBlockEditor = 'modal-cidr-block-editor';
export const ModalContestEditor = 'modal-contest-editor';
export const ModalContestEntry = 'modal-contest-entry';
export const ModalContestEntryDelete = 'modal-contest-entry-delete';
export const ModalConfirm = 'modal-confirm';
export const ModalAssetViewer = 'modal-asset-viewer';
export const ModalBanSteam = 'modal-ban-steam';
export const ModalBanASN = 'modal-ban-asn';
export const ModalBanCIDR = 'modal-ban-cidr';
export const ModalBanGroup = 'modal-ban-group';
export const ModalUnbanSteam = 'modal-unban-steam';
export const ModalUnbanASN = 'modal-unban-asn';
export const ModalUnbanCIDR = 'modal-unban-cidr';
export const ModalUnbanGroup = 'modal-unban-group';
export const ModalServerEditor = 'modal-server-editor';
export const ModalServerDelete = 'modal-server-delete';
export const ModalMessageContext = 'modal-message-context';
export const ModalFileUpload = 'modal-file-upload';
export const ModalFilterEditor = 'modal-filter-editor';
export const ModalPersonEditor = 'modal-person-editor';
export const ModalForumCategoryEditor = 'modal-forum-category-editor';
export const ModalForumForumEditor = 'modal-forum-forum-editor';
export const ModalForumThreadCreator = 'modal-forum-thread-creator';
export const ModalForumThreadEditor = 'modal-forum-thread-editor';

[
    [ModalCIDRWhitelistEditor, CIDRWhitelistEditorModal],
    [ModalCIDRBlockEditor, CIDRBlockEditorModal],
    [ModalForumThreadEditor, ForumThreadEditorModal],
    [ModalForumThreadCreator, ForumThreadCreatorModal],
    [ModalForumForumEditor, ForumForumEditorModal],
    [ModalForumCategoryEditor, ForumCategoryEditorModal],
    [ModalContestEntryDelete, ContestEntryDeleteModal],
    [ModalContestEditor, ContestEditor],
    [ModalContestEntry, ContestEntryModal],
    [ModalAssetViewer, AssetViewer],
    [ModalConfirm, ConfirmationModal],
    [ModalServerEditor, ServerEditorModal],
    [ModalServerDelete, ServerDeleteModal],
    [ModalMessageContext, MessageContextModal],
    [ModalPersonEditor, PersonEditModal],
    [ModalFileUpload, FileUploadModal],
    [ModalFilterEditor, FilterEditModal],
    [ModalBanSteam, BanSteamModal],
    [ModalBanASN, BanASNModal],
    [ModalBanCIDR, BanCIDRModal],
    [ModalBanGroup, BanGroupModal],
    [ModalUnbanSteam, UnbanSteamModal],
    [ModalUnbanASN, UnbanASNModal],
    [ModalUnbanCIDR, UnbanCIDRModal],
    [ModalUnbanGroup, UnbanGroupModal]
].map((value) => {
    NiceModal.register(value[0] as never, value[1] as never);
});
