#!/bin/bash

VPS_USER=root
VPS_HOST=your.vps.ip.here
VPS_PORT=22
LOCAL_PATH="$(pwd)"
REMOTE_PATH="/root/Etl-Analysis-flow"

rsync -avz -e "ssh -p $VPS_PORT" $LOCAL_PATH/ $VPS_USER@$VPS_HOST:$REMOTE_PATH

ssh -p $VPS_PORT $VPS_USER@$VPS_HOST << EOF
  cd $REMOTE_PATH
  export \$(grep -v '^#' .env | xargs)
  tailscale up --authkey=\$TAILSCALE_AUTHKEY
  docker-compose -f docker/docker-compose.backend.yml up -d
EOF
