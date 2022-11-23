export STEAMAPPDIR=./tf-dedicated
export STEAMCMDDIR=./steamcmd
export STEAMAPP=tf
export SRCDS_TOKEN=""
export SRCDS_PORT="27015"
export SRCDS_TV_PORT="27016"
export SRCDS_REGION="0"
export SRCDS_HOSTNAME="Minecraft"
export SRCDS_PW="blahblah"
export SRCDS_STARTMAP="pl_badwater"
export SRCDS_RCONPW="test123"
export SRCDS_IP="192.168.0.72"
export SRCDS_MAXPLAYERS="32"
export STEAMAPPID=232250
export SRCDS_FPSMAX=300
export SRCDS_TICKRATE=66

bash "${STEAMCMDDIR}/steamcmd.sh" +force_install_dir "${STEAMAPPDIR}" \
				+login anonymous \
				+app_update "${STEAMAPPID}" \
				+quit || exit

bash "${STEAMAPPDIR}/srcds_run" -game "${STEAMAPP}" -console -autoupdate \
  -steam_dir "${STEAMCMDDIR}" \
  -steamcmd_script "${HOMEDIR}/${STEAMAPP}_update.txt" \
  -usercon \
  -port "${SRCDS_PORT}" \
  +tv_port "${SRCDS_TV_PORT}" \
  +clientport "${SRCDS_CLIENT_PORT}" \
  +maxplayers "${SRCDS_MAXPLAYERS}" \
  +map "${SRCDS_STARTMAP}" \
  +sv_setsteamaccount "${SRCDS_TOKEN}" \
  +rcon_password "${SRCDS_RCONPW}" \
  +sv_password "${SRCDS_PW}" \
  +sv_region "${SRCDS_REGION}" \
  -ip "${SRCDS_IP}" \
  -authkey "${SRCDS_WORKSHOP_AUTHKEY}"
#   +fps_max "${SRCDS_FPSMAX}" \
#  -tickrate "${SRCDS_TICKRATE}" \
