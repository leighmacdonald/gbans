# scp -i ~/.ssh/pk-putty-nopass.key extensions/rip.ext.so tf2server@$TF2_HOST:~/serverfiles/tf/addons/sourcemod/extensions
cp -v extensions/system2.ext.dll /c/projects/gbans/tf2server/steamapps/common/Team\ Fortress\ 2\ Dedicated\ Server/tf/addons/sourcemod/extensions
mkdir -p plugins
rm -f "../plugins/gbans.smx"
clang-format --assume-filename="gbans.cpp" -i gbans.sp && \
spcomp.exe -v2 -E "gbans.sp" && \
mv "gbans.smx" "plugins/" && \
#scp -i ~/.ssh/pk-putty-nopass.key plugins/gbans.smx tf2server@$TF2_HOST:~/serverfiles/tf/addons/sourcemod/plugins
cp -v plugins/gbans.smx /c/projects/gbans/tf2server/steamapps/common/Team\ Fortress\ 2\ Dedicated\ Server/tf/addons/sourcemod/plugins
rcon -H $TF2_HOST -p $RCON_PASS sm plugins unload gbans
rcon -H $TF2_HOST -p $RCON_PASS sm plugins load gbans
sleep 10