# scp -i ~/.ssh/pk-putty-nopass.key extensions/rip.ext.so tf2server@$TF2_HOST:~/serverfiles/tf/addons/sourcemod/extensions
cp -v extensions/system2.ext.dll /s/tf2server/steamapps/common/Team\ Fortress\ 2\ Dedicated\ Server/tf/addons/sourcemod/extensions
mkdir -p plugins
rm -f "../plugins/gban.smx"
spcomp.exe "gban.sp" && \
mv "gban.smx" "plugins/" && \
#scp -i ~/.ssh/pk-putty-nopass.key plugins/gban.smx tf2server@$TF2_HOST:~/serverfiles/tf/addons/sourcemod/plugins
cp -v plugins/gban.smx /s/tf2server/steamapps/common/Team\ Fortress\ 2\ Dedicated\ Server/tf/addons/sourcemod/plugins
rcon -H $TF2_HOST -p $RCON_PASS sm plugins reload gban
