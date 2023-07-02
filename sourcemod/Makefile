.PHONY: all build fmt copy

SM_HOME=$(HOME)/sdk/sourcemod-1.11.0/addons/sourcemod

all: fmt build

fmt:
	find scripting/ -not -path "./include/*" -iname gbans.inc -o -iname *.sp -type f -exec sp_format {} \;

build:
	spcomp64 scripting/gbans.sp -o plugins/gbans.smx -i$(SM_HOME)/scripting/include -i scripting/include -v2

copy:
	cp -rv scripting/* $(HOME)/projects/uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/
