import { useModal } from "@ebay/nice-modal-react";
import { type ReactNode, useEffect, useState } from "react";
import useWebSocket, { ReadyState } from "react-use-websocket";
import { type QueueRequest, websocketURL } from "../../api";
import { useAuth } from "../../hooks/useAuth.ts";
import { QueueCtx } from "../../hooks/useQueueCtx.ts";
import * as killsound from "../../icons/Killsound.mp3";
import { PermissionLevel } from "../../schema/people.ts";
import {
	type ChatLog,
	type ChatStatus,
	type ChatStatusChangePayload,
	type ClientStatePayload,
	type GameStartPayload,
	type JoinPayload,
	type LeavePayload,
	type LobbyState,
	type MessageCreatePayload,
	type MessagePayload,
	Operation,
	type PurgePayload,
	type QueueMember,
} from "../../schema/playerqueue.ts";
import { readAccessToken } from "../../util/auth/readAccessToken.ts";
import { logErr } from "../../util/errors.ts";
import { transformCreatedOnDate } from "../../util/time.ts";
import { ModalQueueJoin } from "../modal";

/**
 * QueueProvider provides a high level context for server queueing. The intention is to allow users to
 * queue up for a particular server and be able to fill it once some minimum threshold is reached.
 *
 * @param children
 * @constructor
 */
export const QueueProvider = ({ children }: { children: ReactNode }) => {
	const [isReady, setIsReady] = useState(false);
	const [users, setUsers] = useState<QueueMember[]>([]);
	const [messages, setMessages] = useState<ChatLog[]>([]);
	const [showChat, setShowChat] = useState(false);
	const [lobbies, setLobbies] = useState<LobbyState[]>([]);
	const { profile, isAuthenticated } = useAuth();
	const [chatStatus, setChatStatus] = useState<ChatStatus>(profile.playerqueue_chat_status);
	const [reason, setReason] = useState<string>("");

	const modal = useModal(ModalQueueJoin);

	const { readyState, sendJsonMessage, lastJsonMessage } = useWebSocket(websocketURL(), {
		queryParams: { token: isAuthenticated() ? readAccessToken() : "" },
		heartbeat: true,
		share: true,
		//reconnectInterval: 10,
		shouldReconnect: () => true,
	});

	useEffect(() => {
		switch (readyState) {
			case ReadyState.OPEN:
				setIsReady(true);
				setMessages((prevState) => [
					...prevState,
					{
						created_on: new Date(),
						body_md: "Connected to queue",
						avatarhash: "",
						permission_level: PermissionLevel.Reserved,
						personaname: "SYSTEM",
						steam_id: "SYSTEM",
					} as ChatLog,
				]);
				break;
			case ReadyState.CLOSED:
				setIsReady(false);
				setMessages((prevState) => [
					...prevState,
					{
						created_on: new Date(),
						body_md: "Disconnected from queue",
						avatarhash: "",
						permission_level: PermissionLevel.Reserved,
						personaname: "SYSTEM",
						steam_id: "SYSTEM",
					} as ChatLog,
				]);
				break;
			default:
				break;
		}
	}, [readyState]);

	useEffect(() => {
		const request = lastJsonMessage as QueueRequest<never>;
		if (!request) {
			return;
		}
		handleIncomingOperation(request).catch(logErr);
	}, [lastJsonMessage, handleIncomingOperation]);

	const handleIncomingOperation = async (request: QueueRequest<never>) => {
		switch (request.op) {
			case Operation.StateUpdate: {
				updateState(request.payload as ClientStatePayload);
				break;
			}

			case Operation.Message: {
				setMessages((prev) => {
					try {
						const messages = (request.payload as MessagePayload).messages.map(transformCreatedOnDate);
						let all = [...prev, ...messages];
						if (all.length > 100) {
							all = all.slice(all.length - 100, 100);
						}
						return all;
					} catch (e) {
						logErr(e);
						return prev;
					}
				});
				break;
			}

			case Operation.StartGame: {
				await startGame(request.payload as GameStartPayload);
				break;
			}

			case Operation.Purge: {
				purgeMessages((request.payload as PurgePayload).message_ids);
				break;
			}
			case Operation.ChatStatusChange: {
				const pl = request.payload as ChatStatusChangePayload;
				setChatStatus(pl.status);
				if (pl.status === "noaccess") {
					setMessages([]);
				}
				setReason(pl.reason);
			}
		}
	};

	const updateState = (state: ClientStatePayload) => {
		if (state.update_users) {
			setUsers(state.users);
		}
		if (state.update_servers) {
			setLobbies(state.lobbies);
		}
	};

	const purgeMessages = (message_ids: number[]) => {
		setMessages((prevState) => prevState.filter((m) => !message_ids.includes(m.message_id)));
	};

	const startGame = async (gameStart: GameStartPayload) => {
		const audio = new Audio(killsound.default);
		try {
			await audio.play();
		} catch (e) {
			logErr(e);
		}

		await modal.show({ gameStart });
	};

	const sendMessage = (body_md: string) => {
		sendJsonMessage<QueueRequest<MessageCreatePayload>>({
			op: Operation.Message,
			payload: {
				body_md,
			},
		});
	};

	const joinQueue = (servers: string[]) => {
		sendJsonMessage<QueueRequest<JoinPayload>>({
			op: Operation.JoinQueue,
			payload: {
				servers: servers.map(Number),
			},
		});
	};
	const leaveQueue = (servers: string[]) => {
		sendJsonMessage<QueueRequest<LeavePayload>>({
			op: Operation.LeaveQueue,
			payload: {
				servers: servers.map(Number),
			},
		});
	};

	return (
		<QueueCtx.Provider
			value={{
				users,
				lobbies,
				messages,
				isReady,
				sendMessage,
				joinQueue,
				leaveQueue,
				showChat,
				setShowChat,
				chatStatus,
				reason,
			}}
		>
			{children}
		</QueueCtx.Provider>
	);
};
