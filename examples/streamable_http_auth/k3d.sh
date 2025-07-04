#!/bin/bash


# TODO: add caddy to listen on port 5553..

k3d cluster create \
  --k3s-arg "--kube-apiserver-arg=--oidc-issuer-url=https://caddy:443/dex@server:*" \
  --k3s-arg "--kube-apiserver-arg=--oidc-username-claim=email@server:*" \
  --k3s-arg "--kube-apiserver-arg=--oidc-client-id=example-app@server:*" \
  --k3s-arg "--kube-apiserver-arg=--oidc-ca-file=/caddy-pki/intermediate.crt@server:*" \
  --network streamable_http_auth_default \
  --volume "$(pwd -W)/caddy-pki:/caddy-pki@server:*" \
  mcp-test-auth

# curl https://caddy:443/dex/.well-known/openid-configuration