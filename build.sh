#!/bin/bash

#####
# /*
# * build
# */
#####


SERVER_ACCES="mercur@mercur.eftasgmbh.local"
APP_DIR="/usr/app"
PROJECT_DIR="/srv/docker/files/wolweb"


docker-service-restart(){
        docker compose stop $@
        docker compose rm -f -v $@
        docker compose up --no-start --force-recreate $@
        docker compose start $@
}


if [ $# -eq 0 ]; then
    echo "docker tag required"
    exit 0
fi


VERSION=$(git describe --tags --abbrev=0 --always)
VERSION="${VERSION//.}"


echo "Building wolweb image with tag: $1 and version: ${VERSION}"
DOCKER_BUILDKIT=1 docker buildx build --build-arg stage="production" --tag $1 --build-arg version="${VERSION}" -f Dockerfile .  # --debug  --no-cache

if [ $? -ne 0 ]; then
    echo "docker build failed!"
    exit 1
fi

docker push $1

if [[ $VERSION =~ ^"v" ]]; then
    tag=$(echo $1 | sed "s/:dev/:${VERSION}/")
    echo "Pushing $tag"
    docker push $tag
fi
# test $? -eq 0 || echo "something bad happened" && exit

# ssh ${SERVER_ACCES} 'cd /srv/docker/files/cropanalyzer ; docker compose pull; docker compose down --remove-orphans; docker compose up -d ; docker compose logs -f'
