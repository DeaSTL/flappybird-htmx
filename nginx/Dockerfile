FROM nginx:alpine

COPY ./nginx-init.sh ./
COPY ./test-region /environment/region

ARG DOMAIN

RUN apk update
RUN apk add certbot certbot-nginx

ENV DOMAIN=$DOMAIN

RUN ./nginx-init.sh

ENTRYPOINT nginx -g 'daemon off;' && certbot delete --cert-name $DOMAIN


