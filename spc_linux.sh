mkdir -p plugins
rm -f "../plugins/gbans.smx"
clang-format --assume-filename="gbans.cpp" -i gbans.sp && \
spcomp -v2 -E "gbans.sp" && \
mv "gbans.smx" "plugins/" && \
cp -v plugins/gbans.smx ~/serverfiles/tf/addons/sourcemod/plugins
rcon -H $TF2_HOST -p $RCON_PASS sm plugins unload gbans
rcon -H $TF2_HOST -p $RCON_PASS sm plugins load gbans
