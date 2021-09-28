FROM node:16.10.0-stretch-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod a+x ./kubectl && \
    mv ./kubectl /usr/bin/kubectl && \
    kubectl version --client

RUN npm install -g \
      coffeescript \
      generator-hubot \
      yo && \
    mkdir -p /kubot/scripts && \
    chown -R node:node /kubot

RUN mkdir -p /home/node/.kube && \
    chown -R node:node /home/node

USER node
WORKDIR /kubot

RUN yo hubot \
      --owner="KubeVirt <kubevirtbot@redhat.com>" \
      --name="kubot" \
      --description="Manage CI clusters through Slack" \
      --adapter="slack" \
      --defaults && \
    npm install hubot-scripts && \
    npm install yaml --save
COPY worker.coffee /kubot/scripts/

ENV DISABLE_USER_SYNC=true

ENTRYPOINT ["bin/hubot", "--adapter", "slack"]
