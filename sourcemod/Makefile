.PHONY: all build fmt copy

SM_HOME=$(HOME)/sdk/sourcemod-1.11.0/addons/sourcemod

all: fmt build

fmt:
	find scripting/ -iname gbans.inc -o -iname *.sp | xargs clang-format -i --verbose

build:
	spcomp64 scripting/gbans.sp -o plugins/gbans.smx -i$(SM_HOME)/scripting/include -i scripting/include -v2

copy:
	cp -rv scripting/* $(HOME)/projects/uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/
