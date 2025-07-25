import NiceModal from '@ebay/nice-modal-react';
import { AssetViewer } from './AssetViewer.tsx';
import { BanASNModal } from './BanASNModal.tsx';
import { BanCIDRModal } from './BanCIDRModal.tsx';
import { BanGroupModal } from './BanGroupModal.tsx';
import { BanSteamModal } from './BanSteamModal.tsx';
import { CIDRBlockEditorModal } from './CIDRBlockEditorModal.tsx';
import { ConfirmationModal } from './ConfirmationModal.tsx';
import { ContestEditor } from './ContestEditor.tsx';
import { ContestEntryDeleteModal } from './ContestEntryDeleteModal.tsx';
import { ContestEntryModal } from './ContestEntryModal.tsx';
import { FilterEditModal } from './FilterEditModal.tsx';
import { ForumCategoryEditorModal } from './ForumCategoryEditorModal.tsx';
import { ForumForumEditorModal } from './ForumForumEditorModal.tsx';
import { ForumThreadCreatorModal } from './ForumThreadCreatorModal.tsx';
import { ForumThreadEditorModal } from './ForumThreadEditorModal.tsx';
import { IPWhitelistEditorModal } from './IPWhitelistEditorModal.tsx';
import { NewsEditModal } from './NewsEditModal.tsx';
import { PersonEditModal } from './PersonEditModal.tsx';
import { QueueJoinModal } from './QueueJoinModal.tsx';
import { QueueStatusModal } from './QueueStatusModal.tsx';
import { SMAdminEditorModal } from './SMAdminEditorModal.tsx';
import { SMGroupEditorModal } from './SMGroupEditorModal.tsx';
import { SMGroupImmunityCreateModal } from './SMGroupImmunityCreateModal.tsx';
import { SMGroupOverrideEditorModal } from './SMGroupOverrideEditorModal.tsx';
import { SMGroupOverridesModal } from './SMGroupOverridesModal.tsx';
import { SMGroupSelectModal } from './SMGroupSelectModal.tsx';
import { SMOverrideEditorModal } from './SMOverrideEditorModal.tsx';
import { ServerEditorModal } from './ServerEditorModal.tsx';
import { SteamWhitelistEditorModal } from './SteamWhitelistEditorModal.tsx';
import { UnbanASNModal } from './UnbanASNModal.tsx';
import { UnbanCIDRModal } from './UnbanCIDRModal.tsx';
import { UnbanGroupModal } from './UnbanGroupModal.tsx';
import { UnbanSteamModal } from './UnbanSteamModal.tsx';

export const ModalSMGroupImmunityEditor = 'modal-sm-group-immunity-editor';
export const ModalSMGroupOverridesEditor = 'modal-sm-group-overrides-editor';
export const ModalSMOverridesEditor = 'modal-sm-overrides-editor';
export const ModalSMGroupOverrides = 'modal-sm-group-overrides';
export const ModalSMGroupSelect = 'modal-sm-group-select';
export const ModalSMGroupEditor = 'modal-sm-group-editor';
export const ModalSMAdminEditor = 'modal-sm-admin-editor';
export const ModalSteamWhitelistEditor = 'modal-steam-whitelist-editor';
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
export const ModalFilterEditor = 'modal-filter-editor';
export const ModalPersonEditor = 'modal-person-editor';
export const ModalForumCategoryEditor = 'modal-forum-category-editor';
export const ModalForumForumEditor = 'modal-forum-forum-editor';
export const ModalForumThreadCreator = 'modal-forum-thread-creator';
export const ModalForumThreadEditor = 'modal-forum-thread-editor';
export const ModalNewsEditor = 'modal-news-editor';
export const ModalQueueJoin = 'modal-queue-join';
export const ModalQueuePurge = 'modal-queue-delete-messages';
export const ModalQueueStatus = 'modal-queue-status';
[
    //[ModalQueuePurge, QueuePurgeModal],
    [ModalQueueStatus, QueueStatusModal],
    [ModalQueueJoin, QueueJoinModal],
    [ModalSMGroupImmunityEditor, SMGroupImmunityCreateModal],
    [ModalSMGroupOverridesEditor, SMGroupOverrideEditorModal],
    [ModalSMOverridesEditor, SMOverrideEditorModal],
    [ModalSMGroupOverrides, SMGroupOverridesModal],
    [ModalSMGroupSelect, SMGroupSelectModal],
    [ModalSMGroupEditor, SMGroupEditorModal],
    [ModalSMAdminEditor, SMAdminEditorModal],
    [ModalSteamWhitelistEditor, SteamWhitelistEditorModal],
    [ModalCIDRWhitelistEditor, IPWhitelistEditorModal],
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
    [ModalPersonEditor, PersonEditModal],
    [ModalFilterEditor, FilterEditModal],
    [ModalBanSteam, BanSteamModal],
    [ModalBanASN, BanASNModal],
    [ModalBanCIDR, BanCIDRModal],
    [ModalBanGroup, BanGroupModal],
    [ModalUnbanSteam, UnbanSteamModal],
    [ModalUnbanASN, UnbanASNModal],
    [ModalUnbanCIDR, UnbanCIDRModal],
    [ModalUnbanGroup, UnbanGroupModal],
    [ModalNewsEditor, NewsEditModal]
].map((value) => {
    NiceModal.register(value[0] as string, value[1] as never);
});
