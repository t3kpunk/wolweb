#!/bin/bash
# /*
# /  docker compose.yaml
# /
# /  # Testing only!
# /  volumes:
# /    - "/srv/docker/git/wolweb/wolweb:/app/wolweb"
# */

# sudo usermod -aG docker $USER
# newgrp docker
# docker push registry.dev.eftas.com/eftas/wolweb:latest

go build -o wolweb .


#docker build --build-arg stage="production" --tag registry.dev.eftas.com/eftas/wolweb:latest -f /srv/docker/git/wolweb/Dockerfile.dev .

SERVER="mercur.eftasgmbh.local"
USER="mercur"
PROJECT_DIR="/app"
DST_DIR="/srv/docker/files/wolweb/"
BUILD_DIR="/srv/docker/git/wolweb/"
SRC_DIR="/home/wkl/Projects/wolweb/"

function docker-service-restart(){
        docker compose stop $@
        docker compose rm -f -v $@
        docker compose up --no-start --force-recreate $@
        docker compose start $@
}


if [ $# -eq 0 ]; then
    #scp wolweb ${USER}@${SERVER}:${BUILD_DIR}
    rsync -e ssh -avz --delete . ${USER}@${SERVER}:${BUILD_DIR} #--dry-run
fi

ssh ${USER}@${SERVER} 'cd /srv/docker/git/wolweb/ ; docker build --build-arg stage="production" --tag registry.dev.eftas.com/eftas/wolweb:latest -f Dockerfile.dev . ; cd /srv/docker/files/wolweb/ ; docker compose stop wolweb; docker compose up --force-recreate wolweb; docker compose logs wolweb -f'

