scp -i ~/.ssh/pk-putty-nopass.key extensions/rip.ext.so tf2server@172.16.1.100:~/serverfiles/tf/addons/sourcemod/extensions
mkdir -p plugins
rm -f "../plugins/gban.smx"
spcomp.exe "gban.sp" && \
mv "gban.smx" "plugins/" && \
scp -i ~/.ssh/pk-putty-nopass.key plugins/gban.smx tf2server@172.16.1.100:~/serverfiles/tf/addons/sourcemod/plugins
rcon -H 172.16.1.100 -p yepx2 sm plugins reload gban
