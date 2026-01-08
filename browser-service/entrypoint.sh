#!/bin/bash
set -e

# Start Xvfb
Xvfb :99 -screen 0 1920x1080x24 &
sleep 2

# Start fluxbox window manager
fluxbox &
sleep 1

# Start VNC server
x11vnc -display :99 -forever -shared -rfbport 5900 -passwd ${VNC_PASSWORD:-secret123} &
sleep 1

# Start supervisor for browser and controller
exec /usr/bin/supervisord -c /etc/supervisor/supervisord.conf
