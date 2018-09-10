#!/bin/bash

docker tag jenkins-blueocean:lts-alpine alukiano/jenkins-blueocean:lts-alpine
docker push alukiano/jenkins-blueocean:lts-alpine
