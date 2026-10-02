#!/bin/sh
# xrdp session start for the devbox (installed as /etc/xrdp/startwm.sh by the Dockerfile), used by Kasm's
# "Devbox (RDP)" Guacamole workspace and by mstsc on host port 13389. 2026-10-02, P-1.
#
# Why not the stock script: it runs ~/.xsession or falls back to gnome-session (absent here; it aborts without a
# system bus). And the home is shared with root-run KasmVNC desktops, which keep rewriting ~/.ICEauthority as root,
# so xfce4-session in an RDP session (uid 1000) failed "Unable to access ~/.ICEauthority: Permission denied".
# This keeps the ICE authority file private to the RDP session and starts XFCE on its own session bus.
# Byline: Claude Code · Opus 5.5 · 2026-10-02
[ -r /etc/profile ] && . /etc/profile
export ICEAUTHORITY="/tmp/.ICEauthority-xrdp-$(id -u)"
unset DBUS_SESSION_BUS_ADDRESS SESSION_MANAGER
exec dbus-launch --exit-with-session startxfce4
