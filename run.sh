#!/bin/sh -x

TARGET_HOST=${1:-dev-56016661.okta.com}
CLIENT_ID=${2:?Missed CLIENT ID parameter}
CLIENT_SECRET=${3:?Missed CLIENT SECRET parameter...}
USER=${4:?Missed user id. Exiting...}
PWD=${5:?Missed user password. Exiting...}
REDIRECT_HOST=localhost
REDIRECT_PORT=8080
REDIRECT_URI=${6:-http://$REDIRECT_HOST:$REDIRECT_PORT/authorization-code/callback}


killall -9 authz_code 

#
# Recompile just in case
#
go mod init github.com/alex-shekhter-vandy/oauth_authz_code
go get github.com/google/uuid

go build authz_code.go
./authz_code $TARGET_HOST $CLIENT_ID $CLIENT_SECRET $REDIRECT_URI $USER $PWD

