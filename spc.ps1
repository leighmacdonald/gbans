Set-Location -Path .\sourcemod
if(!(Test-Path -Path "..\tf2server\steamapps\common\Team Fortress 2 Dedicated Server\tf\addons\sourcemod\extensions\system2.ext.dll" )){
    Copy-Item -Path ".\extensions\system2.ext.dll" -Destination "..\tf2server\steamapps\common\Team Fortress 2 Dedicated Server\tf\addons\sourcemod\extensions\"
}
if(!(Test-Path -Path ".\plugins" )){
    New-Item -ItemType Directory -Path ".\plugins"
}
if((Test-Path -Path ".\plugins\gbans.smx" )){ 
    Remove-Item -Path "plugins\gbans.smx"
}

Copy-Item -Path ".\adminmenu_custom.txt" -Force -Destination "..\tf2server\steamapps\common\Team Fortress 2 Dedicated Server\tf\addons\sourcemod\configs\"

clang-format --assume-filename="gbans.cpp" -i gbans.sp
if (0 -ne $LastExitCode) {
    Write-Host "Clang-Format failed"
    exit 1
}
spcomp.exe -v2 -E "gbans.sp"
if (0 -ne $LastExitCode) {
    Write-Host "Sourcepawn compilation failed"
    exit 1
}
Move-Item -Path "gbans.smx" -Destination "plugins\" 

if((Test-Path -Path ".\tf2server" )){
    Copy-Item -Path "plugins\gbans.smx" -Destination "..\tf2server\steamapps\common\Team Fortress 2 Dedicated Server\tf\addons\sourcemod\plugins\"
}

scp .\plugins\gbans.smx tf2server@test-1:~/serverfiles/tf/addons/sourcemod/plugins
rcon -H $env:TF2_HOST -p $env:RCON_PASS sm plugins unload gbans
rcon -H $env:TF2_HOST -p $env:RCON_PASS sm plugins load gbans

exit 0