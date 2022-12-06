.PHONY: all build copy

all: build

fmt:
	find scripting/ -iname *.inc -o -iname *.sp | xargs clang-format -i

build:
	spcomp64 scripting/gbans.sp -o plugins/gbans.smx -i$(HOME)/sdk/sourcemod-1.11.0/addons/sourcemod/scripting/include -i scripting/include -i $(HOME)/projects/uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/include/ -v2

copy:
	cp -rv scripting/* $(HOME)/projects/uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/
